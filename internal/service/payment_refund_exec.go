package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"go.mongodb.org/mongo-driver/bson"
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
	// Build PayU Hash
	// Format:
	// key|command|var1|salt
	// -------------------------------
	hashStr := fmt.Sprintf(
		"%s|cancel_refund_transaction|%s|%s",
		s.key,
		mihpayid,
		s.salt,
	)

	// DEBUG HASH PRINT
	log.Println("PAYU HASH STRING:", hashStr)
	log.Println("PAYU HASH VALUE :", sha512Hash(hashStr))

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

	if err != nil {
		return fmt.Errorf("payu refund request failed: %w", err)
	}

	log.Println("PayU refund response:", string(resp.Body()))

	if resp.IsError() {
		return fmt.Errorf(
			"payu refund failed: %s",
			string(resp.Body()),
		)
	}

	// -------------------------------
	// Mark Refund Processing
	// -------------------------------
	return s.repo.UpdateTxn(ctx, mihpayid, bson.M{
		"refundStatus": "processing",
		"refundRef":    refundID,
	})
}
