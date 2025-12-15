package payment

import (
	"context"
	"fmt"
	"strings"
)

func (s *Service) ProcessWebhook(ctx context.Context, post map[string]any) error {
	txnid := fmt.Sprintf("%v", post["txnid"])
	status := fmt.Sprintf("%v", post["status"])
	amount := fmt.Sprintf("%v", post["amount"])
	email := fmt.Sprintf("%v", post["email"])
	firstname := fmt.Sprintf("%v", post["firstname"])
	productinfo := fmt.Sprintf("%v", post["productinfo"])

	hashString := fmt.Sprintf(
		"%s|%s|||||||||||%s|%s|%s|%s|%s|%s",
		s.salt, status, email, firstname, productinfo, amount, txnid, s.key,
	)

	if generateHash(hashString) != fmt.Sprintf("%v", post["hash"]) {
		return fmt.Errorf("hash mismatch")
	}

	newStatus := "failed"
	if strings.ToLower(status) == "success" {
		newStatus = "paid"
	}

	_ = s.repo.UpdateByTxnID(ctx, txnid, map[string]any{
		"status":      newStatus,
		"mihpayid":    post["mihpayid"],
		"method":      post["mode"],
		"txnResponse": post,
	})

	s.notifyCh <- post
	return nil
}
