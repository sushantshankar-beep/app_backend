package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/events"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/socket"
	"math"

	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	// "github.com/aws/aws-sdk-go/service/s3/s3manager"
	// 	"go.mongodb.org/mongo-driver/bson/primitive"
	"app_backend/internal/queue"
	"app_backend/internal/s3"
)

type PaymentService struct {
	repo        *repository.PaymentRepository
	transactionRepo *repository.PaymentRepository
	invoiceSvc  *InvoiceService
	socket      *socket.Emitter
	redis       *redis.Client

	acceptedServiceRepo ports.AcceptedServiceRepository
	userRepo            *repository.UserRepo
	providerRepo        ports.ProviderRepo
	notify              ports.NotificationService
	events              *events.Bus
	key                 string
	salt                string
	payuURL             string
	baseURL             string
	http                *resty.Client
	refundRepo *repository.RefundRepo
	invoiceQueue        *queue.InvoiceQueue
	biddingSvc           *BiddingService
	couponSvc 		  *CouponService
}

func NewPaymentService(
	repo *repository.PaymentRepository,
	invoiceRepo *repository.InvoiceRepo,
	socket *socket.Emitter,
	acceptedRepo ports.AcceptedServiceRepository,
	userRepo *repository.UserRepo,
	providerRepo ports.ProviderRepo,
	notify ports.NotificationService,
	eventsBus *events.Bus,
	key, salt, payuURL, baseURL string,
	redis *redis.Client,
	refundRepo *repository.RefundRepo,
	s3Uploader *s3.InvoiceUploader,
	invoiceQueue        *queue.InvoiceQueue,
	biddingSvc          *BiddingService,
	couponSvc 		  *CouponService,
) *PaymentService {

	return &PaymentService{
		repo:                repo,
		invoiceSvc:          NewInvoiceService(invoiceRepo,acceptedRepo.(*repository.AcceptedServiceRepo),userRepo,providerRepo.(*repository.ProviderRepo),repo,s3Uploader),
		socket:              socket,
		acceptedServiceRepo: acceptedRepo,
		userRepo:			 userRepo,
		providerRepo:        providerRepo,
		notify:              notify,
		events:              eventsBus,
		redis:               redis,
		key:                 key,
		salt:                salt,
		payuURL:             payuURL,
		baseURL:             baseURL,
		http:                resty.New().SetTimeout(30 * time.Second),
		refundRepo: refundRepo,
		invoiceQueue:invoiceQueue,
		biddingSvc: biddingSvc,
		couponSvc:couponSvc,

	}
}


func sha512Hash(input string) string {
	h := sha512.Sum512([]byte(input))
	return hex.EncodeToString(h[:])
}

/* ---------------- INITIATE PAYMENT ---------------- */
var paymentLuaScript = redis.NewScript(`
	local online = redis.call("GET", KEYS[1])
	if online ~= ARGV[1] then
	    return 0
	end

	if redis.call("EXISTS", KEYS[2]) == 1 then
	    return 2
	end

	redis.call("SET", KEYS[2], ARGV[2], "EX", ARGV[3])
	return 1
	`)

func (s *PaymentService) InitiatePayment(
	parentCtx context.Context,
	serviceID,
	userID,
	name,
	phone string,
	price float64,
) (map[string]string, error) {

	// -------------------------------
	// Validate IDs
	// -------------------------------
	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, errors.New("invalid serviceId")
	}

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid userId")
	}

	// -------------------------------
	// Mongo Read (4s timeout)
	// -------------------------------
	dbCtx, cancel := context.WithTimeout(parentCtx, 4*time.Second)
	defer cancel()

	svc, err := s.acceptedServiceRepo.GetByID(dbCtx, serviceOID)
	if err != nil || svc == nil {
		return nil, errors.New("service not found")
	}

	// Ownership validation (VERY IMPORTANT)
	if svc.User != userOID {
		return nil, errors.New("unauthorized payment attempt")
	}

	if svc.Provider == primitive.NilObjectID {
		return nil, errors.New("no provider assigned")
	}

	providerID := svc.Provider.Hex()

	// -------------------------------
	// Atomic Redis Online + Lock
	// -------------------------------
	providerKey := "provider:online:" + providerID
	lockKey := "payment:reserve:" + serviceID
	lockVal := userID + ":" + strconv.FormatInt(time.Now().Unix(), 10)

	redisCtx, cancelRedis := context.WithTimeout(parentCtx, 6*time.Second)
	defer cancelRedis()

	result, err := paymentLuaScript.Run(
		redisCtx,
		s.redis,
		[]string{providerKey, lockKey},
		"1",
		lockVal,
		120,
	).Result()

	if err != nil {
		return nil, err
	}

	switch result.(int64) {
	case 0:
		return nil, errors.New("provider is not available")
	case 2:
		return nil, errors.New("payment already in progress")
	}

	// -------------------------------
	// Coupon Handling
	// -------------------------------
	var appliedPromo *domain.AppliedPromoSummary
	var appliedDiscount *domain.AppliedDiscountSummary
	var serviceAmount float64
	var totalDiscount float64

	if svc.PendingCoupon != nil {
		appliedPromo = svc.PendingCoupon.AppliedPromo
		appliedDiscount = svc.PendingCoupon.AppliedDiscount
		serviceAmount = svc.PendingCoupon.ServiceAmount
		totalDiscount = svc.PendingCoupon.TotalDiscount
	}

	// -------------------------------
	// Prepare Payment Data
	// -------------------------------
	finalAmount := math.Round(price*100) / 100
	amountStr := strconv.FormatFloat(finalAmount, 'f', 2, 64)

	txnid := "TXN_" + serviceID + "_" + strconv.FormatInt(time.Now().UnixMilli(), 10)
	productinfo := "vahanwire service"
	email := "app" + serviceID + "@vahanwire.com"

	var hashBuilder strings.Builder
	hashBuilder.Grow(200)
	hashBuilder.WriteString(s.key)
	hashBuilder.WriteString("|")
	hashBuilder.WriteString(txnid)
	hashBuilder.WriteString("|")
	hashBuilder.WriteString(amountStr)
	hashBuilder.WriteString("|")
	hashBuilder.WriteString(productinfo)
	hashBuilder.WriteString("|")
	hashBuilder.WriteString(name)
	hashBuilder.WriteString("|")
	hashBuilder.WriteString(email)
	hashBuilder.WriteString("|||||||||||")
	hashBuilder.WriteString(s.salt)
	hash := sha512Hash(hashBuilder.String())
	writeCtx, cancelWrite := context.WithTimeout(parentCtx, 8*time.Second)
	defer cancelWrite()

	err = s.repo.CreateTransaction(writeCtx, &domain.PaymentTransaction{
		TxnID:          txnid,
		Amount:         finalAmount,
		Status:         "pending",
		UserID:         userID,
		ServiceID:      serviceID,
		PaymentSource:  "payu",
		ServiceAmount:  serviceAmount,
		AppliedPromo:   appliedPromo,
		TotalDiscount:  totalDiscount,
		AppliedDiscount: appliedDiscount,
	})
	if err != nil {
		_ = s.redis.Del(parentCtx, lockKey).Err()
		return nil, err
	}

	// -------------------------------
	// Response
	// -------------------------------
	return map[string]string{
		"txnid":       txnid,
		"amount":      amountStr,
		"key":         s.key,
		"hash":        hash,
		"email":       email,
		"firstname":   name,
		"productinfo": productinfo,
		"phone":       phone,
		"payuUrl":     s.payuURL + "/_payment",
		"surl":        s.baseURL + "/payment/webhook",
		"furl":        s.baseURL + "/payment/webhook",
	}, nil
}
func classifyPayUFailure(data map[string]string) domain.PaymentFailReason {

	if data["status"] == "userCancelled" {
		return domain.FailUserCancelled
	}

	msg := strings.ToLower(
		data["error_Message"] + " " +
			data["field9"] + " " +
			data["unmappedstatus"],
	)

	switch {
	case strings.Contains(msg, "bank"):
		return domain.FailBankDecline
	case strings.Contains(msg, "timeout"):
		return domain.FailTimeout
	case strings.Contains(msg, "gateway"):
		return domain.FailGateway
	default:
		return domain.FailUnknown
	}
}


func (s *PaymentService) ProcessWebhook(ctx context.Context, data map[string]string) error {
	fmt.Println("🔥 PAYU RAW WEBHOOK DATA ↓↓↓")
	for k, v := range data {
		fmt.Println(k, "=", v)
	}
	txn, err := s.repo.GetByTxnID(ctx, data["txnid"])
	if err != nil {
		return errors.New("transaction not found")
	}
	serviceOID, err := primitive.ObjectIDFromHex(txn.ServiceID)
	if err != nil {
		return err
	}

	svc, err := s.acceptedServiceRepo.GetByID(ctx, serviceOID)
	if err != nil {
		return err
	}

	if txn.Status == "paid" && data["status"] == "success" {
		return nil
	}
	verifyStr := fmt.Sprintf(
		"%s|%s|||||||||||%s|%s|%s|%s|%s|%s",
		s.salt,
		data["status"],
		data["email"],
		data["firstname"],
		data["productinfo"],
		data["amount"],
		data["txnid"],
		data["key"], // USE key from webhook, not config
	)

	calculated := sha512Hash(verifyStr)

	fmt.Println("🔐 VERIFY STRING:", verifyStr)
	fmt.Println("🔐 CALCULATED HASH:", calculated)
	fmt.Println("🔐 PAYU HASH:", data["hash"])

	if calculated != data["hash"] {
		return errors.New("hash verification failed")
	}

	status := "failed"
	if data["status"] == "success" {
		status = "paid"
		go s.afterPaymentSuccess(txn.TxnID)
		go s.notify.SendToUser(
			context.Background(),
			svc.User.Hex(),
			svc.ID.Hex(),
			"Payment Successful",
			"Your payment was successful. Tracking started.",
			map[string]string{
				"serviceId": svc.ID.Hex(),
			},
		)

		if svc.Provider != primitive.NilObjectID {
			go s.notify.SendToProvider(
				context.Background(),
				svc.Provider.Hex(),
				svc.ID.Hex(),
				"Payment Completed",
				"User completed payment. Start the job.",
				map[string]string{
					"serviceId": svc.ID.Hex(),
				},
			)
		}

	} else {
		reason := classifyPayUFailure(data)
		_ = s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":     status,
		"failReason": reason,
	})
		go s.afterPaymentFailed(txn.TxnID)
		go s.notify.SendToUser(
			context.Background(),
			svc.User.Hex(),
			svc.ID.Hex(),
			"Payment Failed",
			"Payment failed. Please retry.",
			map[string]string{
				"serviceId": svc.ID.Hex(),
			},
		)

		if svc.Provider != primitive.NilObjectID {
			go s.notify.SendToProvider(
				context.Background(),
				svc.Provider.Hex(),
				svc.ID.Hex(),
				"Payment Failed",
				"User payment failed.",
				map[string]string{
					"serviceId": svc.ID.Hex(),
				},
			)
		}

	}
	s.redis.Del(ctx, "payment:reserve:"+txn.ServiceID)

	s.repo.SaveWebhook(ctx, txn.TxnID, toMap(data))

	return s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":   status,
		"mihpayid": data["mihpayid"],
		"method":   data["mode"],
	})
}
func (s *PaymentService) VerifyPayment(
	ctx context.Context,
	txnID string,
) (map[string]any, error) {

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return nil, errors.New("no payment found")
	}
	ttl := s.getRetryTTL(ctx, txnID)

	reason := domain.PaymentFailReason(txn.FailReason)

	resp := map[string]any{
		"serviceId": txn.ServiceID,
		"txnid":     txn.TxnID,
		"amount":    txn.Amount,
		"status":    txn.Status,
		"reason":    txn.FailReason,
		"retryTtl":  ttl,
		"userMsg":   userMessageForReason(reason),
	}

	return resp, nil
}
func userMessageForReason(r domain.PaymentFailReason) string {

	switch r {
	case domain.FailBankDecline:
		return "Bank declined the transaction. Try again or use another method."
	case domain.FailUserCancelled:
		return "You cancelled the payment."
	case domain.FailTimeout:
		return "Payment timed out. Retry quickly."
	case domain.FailGateway:
		return "Payment gateway error. Please retry."
	default:
		return "Payment failed. Try again."
	}
}
func (s *PaymentService) getRetryTTL(ctx context.Context, serviceID string) int {

	key := "payment:reserve:" + serviceID

	ttl, err := s.redis.TTL(ctx, key).Result()
	if err != nil || ttl < 0 {
		return 0
	}

	return int(ttl.Seconds())
}


func (s *PaymentService) Refund(ctx context.Context, mihpayid string, amount float64) error {
	if mihpayid == "" || amount <= 0 {
		return errors.New("invalid refund request")
	}
	job := domain.RefundJob{
		MihPayID: mihpayid,
		Amount:   amount,
		Retries:  0,
	}
	payload, _ := json.Marshal(job)
	if err := s.redis.RPush(ctx, "refund:queue", payload).Err(); err != nil {
		return err
	}
	return s.repo.UpdateTxn(ctx, mihpayid, bson.M{
		"status": "refund_queued",
	})
}
func toMap(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}



func (s *PaymentService) GetRefundTracking(
	ctx context.Context,
	serviceID string,
) (map[string]any, error) {

	refund, err := s.refundRepo.FindByServiceID(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	timeline := []map[string]any{
		{
			"title": "Booking Cancelled",
			"time": refund.CreatedAt,
			"completed": true,
		},
	}

	// under process
	if refund.Status != "" {
		timeline = append(timeline, map[string]any{
			"title": "Refund Under Process",
			"time": refund.UpdatedAt,
			"completed": refund.Status != "failed",
		})
	}

	// final state
	final := map[string]any{
		"title": "Refunded",
		"time": nil,
		"completed": false,
	}

	if refund.Status == "success" {
		final["completed"] = true
		final["time"] = refund.UpdatedAt
	}

	if refund.Status == "failed" {
		final["title"] = "Refund Failed"
		final["completed"] = true
		final["time"] = refund.UpdatedAt
	}

	timeline = append(timeline, final)

	return map[string]any{
		"serviceId": refund.ServiceID,
		"amount": refund.Amount,
		"estimatedDays": "5–7 working days",
		"status": refund.Status,
		"timeline": timeline,
	}, nil
}
