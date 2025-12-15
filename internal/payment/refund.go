package payment

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s *Service) Refund(ctx context.Context, mihpayid string, amount float64) error {
	command := "cancel_refund_transaction"

	hash := generateHash(fmt.Sprintf(
		"%s|%s|%s|%s",
		s.key, command, mihpayid, s.salt,
	))

	form := url.Values{}
	form.Set("key", s.key)
	form.Set("command", command)
	form.Set("hash", hash)
	form.Set("var1", mihpayid)
	form.Set("var3", fmt.Sprintf("%.2f", amount))

	req, _ := http.NewRequest(
		http.MethodPost,
		s.baseURL+"/merchant/postservice?form=2",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	_, err := s.httpClient.Do(req)
	return err
}
