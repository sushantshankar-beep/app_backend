package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *PaymentService) ProcessRefund(
	ctx context.Context,
	mihpayid string,
	amount float64,
) error {

	hashStr := fmt.Sprintf(
		"%s|cancel_refund_transaction|%s|%s",
		s.key,
		mihpayid,
		s.salt,
	)

	form := url.Values{}
	form.Set("key", s.key)
	form.Set("command", "cancel_refund_transaction")
	form.Set("hash", sha512Hash(hashStr))
	form.Set("var1", mihpayid)
	form.Set("var3", fmt.Sprintf("%.2f", amount))

	resp, err := s.http.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(form.Encode()).
		Post(s.payuURL + "/merchant/postservice?form=2")

	if err != nil || resp.IsError() {
		return errors.New("payu refund failed")
	}

	// Update DB on success
	return s.repo.UpdateTxn(ctx, mihpayid, bson.M{
		"status": "refunded",
	})
}
