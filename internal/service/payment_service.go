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
)

type PaymentService struct {
	repo *repository.PaymentRepository

	socket *socket.Emitter
	redis  *redis.Client

	acceptedServiceRepo ports.AcceptedServiceRepo
	providerRepo        ports.ProviderRepo
	notify              ports.NotificationService

	key     string
	salt    string
	payuURL string
	baseURL string
	http    *resty.Client
}

func NewPaymentService(
	repo *repository.PaymentRepository,
	socket *socket.Emitter,
	acceptedServiceRepo ports.AcceptedServiceRepo,
	providerRepo ports.ProviderRepo,
	notify ports.NotificationService,
	key, salt, payuURL, baseURL string,
	redis *redis.Client,
) *PaymentService {
	return &PaymentService{
		repo: repo,
		socket: socket,
		acceptedServiceRepo: acceptedServiceRepo,
		providerRepo: providerRepo,
		notify: notify,
		redis: redis,
		key: key,
		salt: salt,
		payuURL: payuURL,
		baseURL: baseURL,
		http: resty.New().SetTimeout(30 * time.Second),
	}
}

func sha512Hash(input string) string {
	h := sha512.Sum512([]byte(input))
	return hex.EncodeToString(h[:])
}

/* ---------------- INITIATE PAYMENT ---------------- */

func (s *PaymentService) InitiatePayment(
	ctx context.Context,
	serviceID, userID, name, email, phone string,
	price float64,
) (map[string]string, error) {

	// 🔐 Prevent duplicate payment attempts
	lockKey := "payment:reserve:" + serviceID
	ok, err := s.redis.SetNX(ctx, lockKey, "locked", 10*time.Minute).Result()
	if err != nil || !ok {
		return nil, errors.New("service already reserved")
	}

	txnid := fmt.Sprintf("TXN_%s_%d", serviceID, time.Now().UnixMilli())
	amount := strconv.FormatFloat(price*1.18, 'f', 2, 64)

	hashStr := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|||||||||||%s",
		s.key, txnid, amount, serviceID, name, email, s.salt,
	)

	err = s.repo.CreateTransaction(ctx, &domain.PaymentTransaction{
		TxnID:         txnid,
		Amount:        price * 1.18,
		Status:        "pending",
		UserID:        userID,
		ServiceID:     serviceID,
		PaymentSource: "payu",
	})
	if err != nil {
		s.redis.Del(ctx, lockKey)
		return nil, err
	}

	return map[string]string{
		"txnid":   txnid,
		"amount":  amount,
		"key":     s.key,
		"hash":    sha512Hash(hashStr),
		"payuUrl": s.payuURL + "/_payment",
		"surl":    s.baseURL + "/api/payment/webhook",
		"furl":    s.baseURL + "/api/payment/webhook",
	}, nil
}

/* ---------------- WEBHOOK ---------------- */

func (s *PaymentService) ProcessWebhook(ctx context.Context, data map[string]string) error {
	txn, err := s.repo.GetByTxnID(ctx, data["txnid"])
	if err != nil {
		return errors.New("transaction not found")
	}

	// 🛑 Idempotency guard
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

	// 🔓 release reservation
	s.redis.Del(ctx, "payment:reserve:"+txn.ServiceID)

	s.repo.SaveWebhook(ctx, txn.TxnID, toMap(data))

	return s.repo.UpdateTxn(ctx, txn.TxnID, bson.M{
		"status":   status,
		"mihpayid": data["mihpayid"],
		"method":   data["mode"],
	})
}

/* ---------------- REFUND ---------------- */

func (s *PaymentService) Refund(ctx context.Context, mihpayid string, amount float64) error {
	if mihpayid == "" || amount <= 0 {
		return errors.New("invalid refund request")
	}

	job := domain.RefundJob{
		MihPayID: mihpayid,
		Amount:  amount,
		Retries: 0,
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
