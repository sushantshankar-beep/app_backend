package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
)

func (s *PaymentService) ProcessRefund(
	ctx context.Context,
	mihpayid string,
	amount float64,
) (*domain.PayURefundResponse, error) {

	refundID := fmt.Sprintf(
		"RFVH_%s_%d",
		mihpayid,
		time.Now().Unix(),
	)

	hashStr := fmt.Sprintf(
		"%s|cancel_refund_transaction|%s|%s",
		s.key,
		mihpayid,
		s.salt,
	)

	hash := sha512Hash(hashStr)

	form := url.Values{}
	form.Set("key", s.key)
	form.Set("command", "cancel_refund_transaction")
	form.Set("hash", hash)
	form.Set("var1", mihpayid)
	form.Set("var2", refundID)
	form.Set("var3", fmt.Sprintf("%.2f", amount))

	resp, err := s.http.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(form.Encode()).
		Post(s.payuURL + "/merchant/postservice?form=2")

	if err != nil {
		return nil, fmt.Errorf("payu refund request failed: %w", err)
	}

	body := resp.Body()

	log.Println("PayU refund response:", string(body))

	// -------------------------------
	// Parse JSON
	// -------------------------------
	var payuResp domain.PayURefundResponse
	if err := json.Unmarshal(body, &payuResp); err != nil {
		return nil, fmt.Errorf("failed to parse payu response: %w", err)
	}

	// -------------------------------
	// Gateway Error Handling
	// -------------------------------
	if resp.IsError() || payuResp.Status != 1 {
		return &payuResp, fmt.Errorf("payu refund failed: %s", payuResp.Msg)
	}

	// -------------------------------
	// Update DB
	// -------------------------------
	if err := s.repo.UpdateTxn(ctx, mihpayid, bson.M{
		"refundStatus": "processing",
		"refundRef":    refundID,
		"refundReqId":  payuResp.RequestID,
		"updatedAt":    time.Now(),
	}); err != nil {

		return &payuResp, err
	}

	return &payuResp, nil
}
