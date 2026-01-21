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

func (s *PaymentService) InitiatePayment(ctx context.Context, serviceID, userID, name, email, phone string, price float64) (map[string]string, error) {
	lockKey := "payment:reserve:" + serviceID
	lockVal := userID + ":" + strconv.FormatInt(time.Now().Unix(), 10)

	ok, err := s.redis.SetNX(ctx, lockKey, lockVal, 5*time.Minute).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("service already reserved")
	}
	finalAmount := price * 1.18
	amountStr := strconv.FormatFloat(finalAmount, 'f', 2, 64)
	txnid := fmt.Sprintf("TXN_%s_%d", serviceID, time.Now().UnixMilli())
	hashStr := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|||||||||||%s",
		s.key,
		txnid,
		amountStr,
		serviceID,
		name,
		email,
		s.salt,
	)
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
		"amount":  amountStr,
		"key":     s.key,
		"hash":    sha512Hash(hashStr),
		"payuUrl": s.payuURL + "/_payment",
		"surl":    s.baseURL + "/api/payment/webhook",
		"furl":    s.baseURL + "/api/payment/webhook",
	}, nil
}

func (s *PaymentService) ProcessWebhook(ctx context.Context, data map[string]string) error {
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
	
	if sha512Hash(verifyStr) != data["hash"] {
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
