package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s *Service) VerifyPayment(ctx context.Context, txnid string) (map[string]any, error) {
	command := "verify_payment"
	hash := generateHash(fmt.Sprintf("%s|%s|%s|%s", s.key, command, txnid, s.salt))

	form := url.Values{}
	form.Set("key", s.key)
	form.Set("command", command)
	form.Set("var1", txnid)
	form.Set("hash", hash)

	req, _ := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/merchant/postservice.php?form=2",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	details := result["transaction_details"].(map[string]any)
	tx := details[txnid].(map[string]any)

	status := strings.ToLower(fmt.Sprintf("%v", tx["status"]))
	if status == "success" {
		status = "paid"
	}

	_ = s.repo.UpdateByTxnID(ctx, txnid, map[string]any{
		"status":      status,
		"mihpayid":    tx["mihpayid"],
		"method":      tx["mode"],
		"txnResponse": tx,
	})

	return map[string]any{"success": true, "transaction": tx}, nil
}
