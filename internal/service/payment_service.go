package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/repository"
	"app_backend/internal/socket"
	"github.com/go-resty/resty/v2"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"app_backend/internal/events"
	// "go.mongodb.org/mongo-driver/bson/primitive"
	"math"
)

type PaymentService struct {
	repo        *repository.PaymentRepository
	invoiceSvc  *InvoiceService
	socket      *socket.Emitter
	redis       *redis.Client

	acceptedServiceRepo ports.AcceptedServiceRepository
	providerRepo        ports.ProviderRepo
	notify              ports.NotificationService
	events              *events.Bus
	key                 string
	salt                string
	payuURL             string
	baseURL             string
	http                *resty.Client
}

func NewPaymentService(
	repo *repository.PaymentRepository,
	invoiceRepo *repository.InvoiceRepo,
	socket *socket.Emitter,
	acceptedRepo ports.AcceptedServiceRepository,
	providerRepo ports.ProviderRepo,
	notify ports.NotificationService,
	eventsBus *events.Bus,
	key, salt, payuURL, baseURL string,
	redis *redis.Client,
) *PaymentService {

	return &PaymentService{
		repo:                repo,
		invoiceSvc:          NewInvoiceService(invoiceRepo),
		socket:              socket,
		acceptedServiceRepo: acceptedRepo,
		providerRepo:        providerRepo,
		notify:              notify,
		events:              eventsBus,
		redis:               redis,
		key:                 key,
		salt:                salt,
		payuURL:             payuURL,
		baseURL:             baseURL,
		http:                resty.New().SetTimeout(30 * time.Second),
	}
}


func sha512Hash(input string) string {
	h := sha512.Sum512([]byte(input))
	return hex.EncodeToString(h[:])
}

/* ---------------- INITIATE PAYMENT ---------------- */

func (s *PaymentService) InitiatePayment(
	ctx context.Context,
	serviceID,
	userID,
	name,
	phone string,
	price float64,
) (map[string]string, error) {
	email := fmt.Sprintf("app%s@vahanwire.com",serviceID)
	fmt.Println("this is email",email)

	// 🔒 Lock service
	lockKey := "payment:reserve:" + serviceID
	lockVal := userID + ":" + strconv.FormatInt(time.Now().Unix(), 10)

	ok, err := s.redis.SetNX(ctx, lockKey, lockVal, 2*time.Minute).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("payment already in progress")
	}

	PAYU_KEY := s.key
	PAYU_SALT := s.salt

	firstname := name
	finalAmount := math.Round(price*1.18*100) / 100
	amount := fmt.Sprintf("%.2f", finalAmount)

	txnid := fmt.Sprintf("TXN_%s_%d", serviceID, time.Now().UnixMilli())

	productinfo := "vahanwire service"

	// 🔐 PAYU HASH STRING
	hashStr := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|||||||||||%s",
		PAYU_KEY,
		txnid,
		amount,
		productinfo,
		firstname,
		email,
		PAYU_SALT,
	)

	hash := sha512Hash(hashStr)

	// 💾 Save transaction
	if err := s.repo.CreateTransaction(ctx, &domain.PaymentTransaction{
		TxnID:         txnid,
		Amount:        finalAmount,
		Status:        "pending",
		UserID:        userID,
		ServiceID:     serviceID,
		PaymentSource: "payu",
	}); err != nil {
		_ = s.redis.Del(ctx, lockKey).Err()
		return nil, err
	}

	return map[string]string{
		"txnid":   txnid,
		"amount":  amount,
		"key":     PAYU_KEY,
		"hash":    hash,
		"email":      email,
		"firstname":  firstname,
		"productinfo": productinfo,
		"phone":      phone,
		"payuUrl": s.payuURL + "/_payment",
		"surl":    s.baseURL + "/payment/webhook",
		"furl":    s.baseURL + "/payment/webhook",
	}, nil
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
		s.key,
	)

	calculated := sha512Hash(verifyStr)

	fmt.Println("VERIFY STRING:", verifyStr)
	fmt.Println("CALCULATED HASH:", calculated)
	fmt.Println("PAYU HASH:", data["hash"])

	if calculated != data["hash"] {
		return errors.New("hash verification failed")
	}

	status := "failed"
	if data["status"] == "success" {
		status = "paid"
		go s.afterPaymentSuccess(txn.TxnID)
	} else {
		go s.afterPaymentFailed(txn.TxnID)
	}
	s.redis.Del(ctx, "payment:reserve:"+txn.ServiceID)

	s.repo.SaveWebhook(ctx, txn.TxnID, toMap(data))

	return s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":   status,
		"mihpayid": data["mihpayid"],
		"method":   data["mode"],
	})
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
