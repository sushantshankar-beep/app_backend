package service

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"log"
)

func (s *PaymentService) ProcessRefund(
	ctx context.Context,
	mihpayid string,
	amount float64,
) error {

	// -------------------------------
	// Generate Refund ID (RFVH...)
	// -------------------------------
	refundID := fmt.Sprintf(
		"RFVH_%s_%d",
		mihpayid,
		time.Now().Unix(),
	)

	// -------------------------------
	// Build Hash String
	// Format:
	// key|command|var1|var2|var3|salt
	// -------------------------------
	hashStr := fmt.Sprintf(
		"%s|cancel_refund_transaction|%s|%s|%.2f|%s",
		s.key,
		mihpayid,
		refundID,
		amount,
		s.salt,
	)

	// -------------------------------
	// Build Form Body
	// -------------------------------
	form := url.Values{}
	form.Set("key", s.key)
	form.Set("command", "cancel_refund_transaction")
	form.Set("hash", sha512Hash(hashStr))

	form.Set("var1", mihpayid)
	form.Set("var2", refundID)
	form.Set("var3", fmt.Sprintf("%.2f", amount))

	// -------------------------------
	// Call PayU API
	// -------------------------------
	resp, err := s.http.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(form.Encode()).
		Post(s.payuURL + "/merchant/postservice?form=2")
	log.Println("PayU refund response:", string(resp.Body()))
	if err != nil {
		return fmt.Errorf("payu refund request failed: %w", err)
	}

	if resp.IsError() {
		return fmt.Errorf(
			"payu refund failed: %s",
			string(resp.Body()),
		)
	}

	// -------------------------------
	// Mark Refund Processing
	// (Webhook is source of truth)
	// -------------------------------
	return s.repo.UpdateTxn(ctx, mihpayid, bson.M{
		"refundStatus": "processing",
		"refundRef":    refundID,
	})
}
