package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"regexp"
	"log"
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
type PaymentJob struct {
	TxnID string
	Type  string 
}

type PaymentWorkerPool struct {
	queue chan PaymentJob
}
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
	paymentPool *PaymentWorkerPool
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

/* =========== payment worker ============= */

func (s *PaymentService) StartPaymentWorkers(n int) {

	s.paymentPool = &PaymentWorkerPool{
		queue: make(chan PaymentJob, 1000), // buffer
	}

	for i := 0; i < n; i++ {
		go s.paymentWorker(i, s.paymentPool.queue)
	}
}

func (s *PaymentService) paymentWorker(id int, jobs <-chan PaymentJob) {

	for job := range jobs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("🔥 payment worker panic recovered:", r)
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			lockKey := "payment:job:lock:" + job.TxnID

			ok, _ := s.redis.SetNX(ctx, lockKey, "1", 5*time.Minute).Result()
			if !ok {
				return // already processed
			}

			var err error
			for i := 0; i < 3; i++ {

				switch job.Type {
				case "success":
					err = s.afterPaymentSuccess(job.TxnID)

				case "failed":
					err = s.afterPaymentFailed(job.TxnID)
				}

				if err == nil {
					return 
				}

				fmt.Println("retry payment job:", job.TxnID, "attempt:", i+1)

				time.Sleep(time.Duration(i+1) * time.Second)
			}
			log.Println("payment job failed permanently:", job.TxnID)
		}()
	}
}

func (s *PaymentService) enqueuePaymentJob(txnID string, typ string) {
	if s.paymentPool == nil {
		log.Println("payment pool not initialized")
		return
	}
	select {
	case s.paymentPool.queue <- PaymentJob{
		TxnID: txnID,
		Type:  typ,
	}:
	default:
		log.Println("entering default payment queue full:", txnID)
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
	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, errors.New("invalid serviceId")
	}
	fmt.Println("this is user id",userID)

	// userOID, err := primitive.ObjectIDFromHex(userID)
	// if err != nil {
	// 	return nil, errors.New("invalid userId")
	// }
	dbCtx, cancel := context.WithTimeout(parentCtx, 4*time.Second)
	defer cancel()

	svc, err := s.acceptedServiceRepo.GetByID(dbCtx, serviceOID)
	if err != nil || svc == nil {
		return nil, errors.New("service not found")
	}
	// if svc.User != userOID {
	// 	return nil, errors.New("unauthorized payment attempt")
	// }

	userID = svc.User.Hex()
	
	if svc.Provider == primitive.NilObjectID {
		return nil, errors.New("no provider assigned")
	}
	providerID := svc.Provider.Hex()
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
	for k, v := range data {
		fmt.Println(k, "=", v)
	}
	txn, err := s.repo.GetByTxnID(ctx, data["txnid"])
	if err != nil {
		return errors.New("transaction not found")
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
		data["key"],
	)

	calculated := sha512Hash(verifyStr)
	if calculated != data["hash"] {
		return errors.New("hash verification failed")
	}

	status := "failed"
	if data["status"] == "success" {
		status = "paid"
		s.enqueuePaymentJob(txn.TxnID, "success")
	} else {
		reason := classifyPayUFailure(data)
		_ = s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":     status,
		"failReason": reason,
	})
		s.enqueuePaymentJob(txn.TxnID, "failed")
	}
	s.redis.Del(ctx, "payment:reserve:"+txn.ServiceID)

	s.repo.SaveWebhook(ctx, txn.TxnID, toMap(data))

	return s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":   status,
		"mihpayid": data["mihpayid"],
		"method":   data["mode"],
	})
}
func (s *PaymentService) verifyWithPayU(txnID string) (map[string]string, error) {

	fmt.Println("🔍 VERIFY WITH PAYU CALLING FOR TXN:", txnID)

	key := strings.TrimSpace(s.key)
	salt := strings.TrimSpace(s.salt)

	hashStr := fmt.Sprintf("%s|verify_payment|%s|%s", key, txnID, salt)
	hash := sha512Hash(hashStr)

	fmt.Println("🔐 HASH STRING:", hashStr)
	fmt.Println("🔐 HASH:", hash)

	resp, err := s.http.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"key":     key,
			"command": "verify_payment",
			"var1":    txnID,
			"hash":    hash,
		}).
		Post("https://info.payu.in/merchant/postservice.php")

	if err != nil {
		fmt.Println("❌ PAYU REQUEST ERROR:", err)
		return nil, err
	}

	body := string(resp.Body())

	fmt.Println("📦 PAYU RAW RESPONSE ↓↓↓")
	fmt.Println(body)

	result := map[string]string{}

	/* ---------------------------------------------------
	   PHP SERIALIZED RESPONSE
	--------------------------------------------------- */

	if strings.HasPrefix(body, "a:") {

		fmt.Println("⚠ PHP SERIALIZED RESPONSE DETECTED")

		if strings.Contains(body, `"status";s:7:"success"`) {
			result["status"] = "success"
		}

		if strings.Contains(body, `"status";s:6:"failed"`) {
			result["status"] = "failure"
		}

		if strings.Contains(body, `"status";s:7:"pending"`) {
			result["status"] = "pending"
		}

		// Extract mihpayid
		re := regexp.MustCompile(`mihpayid";s:\d+:"(\d+)"`)
		match := re.FindStringSubmatch(body)
		if len(match) > 1 {
			result["mihpayid"] = match[1]
		}

		// Extract payment mode
		reMode := regexp.MustCompile(`mode";s:\d+:"([A-Z]+)"`)
		matchMode := reMode.FindStringSubmatch(body)
		if len(matchMode) > 1 {
			result["mode"] = matchMode[1]
		}

		// Extract bank_ref_num
		reBank := regexp.MustCompile(`bank_ref_num";s:\d+:"(\d+)"`)
		matchBank := reBank.FindStringSubmatch(body)
		if len(matchBank) > 1 {
			result["bank_ref_num"] = matchBank[1]
		}

		fmt.Println("✅ PARSED PHP RESULT:", result)

		return result, nil
	}

	/* ---------------------------------------------------
	   JSON RESPONSE
	--------------------------------------------------- */

	var parsed map[string]any

	if err := json.Unmarshal(resp.Body(), &parsed); err != nil {
		fmt.Println("❌ JSON PARSE ERROR:", err)
		return nil, err
	}

	fmt.Println("📦 PARSED JSON:", parsed)

	txns, ok := parsed["transaction_details"].(map[string]any)
	if !ok {
		return nil, errors.New("transaction_details missing")
	}

	txn, ok := txns[txnID].(map[string]any)
	if !ok {
		return nil, errors.New("txn not found in PayU response")
	}

	result["status"] = fmt.Sprintf("%v", txn["status"])
	result["mihpayid"] = fmt.Sprintf("%v", txn["mihpayid"])
	result["mode"] = fmt.Sprintf("%v", txn["mode"])
	result["bank_ref_num"] = fmt.Sprintf("%v", txn["bank_ref_num"])

	fmt.Println("✅ FINAL VERIFY RESULT:", result)

	return result, nil
}
func (s *PaymentService) VerifyPayment(
	ctx context.Context,
	txnID string,
) (map[string]any, error) {

	txn, err := s.repo.GetByTxnID(ctx, txnID)
	if err != nil {
		return nil, errors.New("no payment found")
	}

	// ------------------------------------------------
	// If transaction still pending → verify with PayU
	// ------------------------------------------------

	if txn.Status == "pending" {

		result, err := s.verifyWithPayU(txnID)

		if err == nil {

			status := strings.ToLower(result["status"])

			// ----------------------------------------
			// SUCCESS CASE
			// ----------------------------------------

			if status == "success" {

				update := bson.M{
					"status": "paid",
				}

				if v := result["mihpayid"]; v != "" {
					update["mihpayid"] = v
				}

				if v := result["mode"]; v != "" {
					update["method"] = v
				}

				if v := result["bank_ref_num"]; v != "" {
					update["bank_ref_num"] = v
				}

				err = s.repo.UpdateTxn(ctx, txnID, update)
				if err != nil {
					return nil, err
				}

				// run success flow
				go s.afterPaymentSuccess(txnID)

				txn.Status = "paid"
			}

			// ----------------------------------------
			// FAILURE CASE
			// ----------------------------------------

			if status == "failure" {

				err = s.repo.UpdateTxn(ctx, txnID, bson.M{
					"status": "failed",
				})

				if err == nil {
					go s.afterPaymentFailed(txnID)
				}

				txn.Status = "failed"
			}
		}
	}

	// ------------------------------------------------
	// Retry TTL
	// ------------------------------------------------

	ttl := s.getRetryTTL(ctx, txn.ServiceID)

	return map[string]any{
		"serviceId": txn.ServiceID,
		"txnid":     txn.TxnID,
		"amount":    txn.Amount,
		"status":    txn.Status,
		"retryTtl":  ttl,
	}, nil
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