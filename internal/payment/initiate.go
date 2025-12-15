package payment

import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"
)

func (s *Service) InitiatePayment(ctx context.Context, service map[string]any) (map[string]any, error) {
	user := service["user"].(map[string]any)

	txnid := fmt.Sprintf("TXN_%v_%d", service["_id"], time.Now().UnixMilli())

	amountFloat, err := parseAmount(fmt.Sprintf("%v", service["finalPrice"]))
	if err != nil {
		return nil, err
	}

	amount := fmt.Sprintf("%.2f", amountFloat*1.18)

	firstname := fmt.Sprintf("%v", user["name"])
	email := fmt.Sprintf("%v", user["email"])
	phone := fmt.Sprintf("%v", user["phone"])
	productinfo := fmt.Sprintf("%v", service["_id"])

	hashString := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|||||||||||%s",
		s.key, txnid, amount, productinfo, firstname, email, s.salt,
	)

	tx := &domain.Transaction{
		TxnID:         txnid,
		Amount:        amountFloat * 1.18,
		Status:        "pending",
		UserID:        toObjectID(user["_id"]),
		ServiceID:     toObjectID(service["_id"]),
		PaymentSource: "payu",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, err
	}

	return map[string]any{
		"txnid":     txnid,
		"amount":    amount,
		"key":       s.key,
		"hash":      generateHash(hashString),
		"productinfo": productinfo,
		"firstname": firstname,
		"email":     email,
		"phone":     phone,
		"payuUrl":   s.baseURL + "/_payment",
		"surl":      s.baseAppURL + "/api/payment/webhook/success",
		"furl":      s.baseAppURL + "/api/payment/webhook/failure",
	}, nil
}
