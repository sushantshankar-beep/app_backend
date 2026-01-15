package sms

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type SMS struct {
	UserID               string
	Password             string
	PEID                 string
	UserTemplateID       string
	ProviderTemplateID   string
	UserSender           string
	ProviderSender       string
}


func SmsTrigger() *SMS {
	return &SMS{
		UserID:             os.Getenv("MUSER_ID"),
		Password:           os.Getenv("MUSERPWD"),
		PEID:               os.Getenv("MUSER_PEID"),

		UserTemplateID:     os.Getenv("MUSER_TEMPLATE_ID"),
		ProviderTemplateID: os.Getenv("MPROVIDER_TEMPLATE_ID"),

		UserSender:         os.Getenv("MUSER_SENDER"),
		ProviderSender:     os.Getenv("MPROVIDER_SENDER"),
	}
}


func (s *SMS) SendOTP(ctx context.Context, phone, otp, role string) error {
	var (
		message    string
		templateID string
		sender     string
	)

	switch role {
	case "provider":
		message = fmt.Sprintf(
			"Dear Vahanwire provider, %s is your OTP for phone verification, do not share it with anyone. -Vahanwire Teachnologies Pvt Ltd",
			otp,
		)
		templateID = s.ProviderTemplateID
		sender = s.ProviderSender

	default:
		message = fmt.Sprintf(
			"Dear Vahanwire user, %s is your OTP for phone verification, do not share it with anyone. -Vahanwire Teachnologies Pvt Ltd",
			otp,
		)
		templateID = s.UserTemplateID
		sender = s.UserSender
	}

	apiURL := fmt.Sprintf(
		"https://myinboxmedia.in/api/mim/SendSMS?userid=%s&pwd=%s&mobile=%s&sender=%s&msgtype=33&msg=%s&peid=%s&templateid=%s",
		s.UserID,
		s.Password,
		phone,
		sender,
		url.QueryEscape(message),
		s.PEID,
		templateID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("SMS API failed: %s", string(body))
	}

	log.Printf("✅ OTP sent | role=%s | phone=%s | msgId=%s\n", role, phone, string(body))
	return nil
}
