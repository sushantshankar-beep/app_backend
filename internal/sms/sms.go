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
	UserID     string
	Password   string
	PEID       string
	TemplateID string
	Sender     string
}

func SmsTrigger() *SMS {
	return &SMS{
		UserID:     os.Getenv("MUSER_ID"),
		Password:   os.Getenv("MUSERPWD"),
		PEID:       os.Getenv("MUSER_PEID"),
		TemplateID: os.Getenv("MPROVIDER_TEMPLATE_ID"),
		Sender:     os.Getenv("MPROVIDER_SENDER"),
	}
}

func (s *SMS) SendOTP(ctx context.Context, phone, msg string) error {
	otp := msg
	message := fmt.Sprintf(
		"Dear Vahanwire provider %s is your OTP for phone verification, do not share it with anyone. - Vahanwire Technologies Pvt Ltd",
		otp,
	)
	url := fmt.Sprintf(
		"https://myinboxmedia.in/api/mim/SendSMS?userid=%s&pwd=%s&mobile=%s&sender=%s&msgtype=33&msg=%s&peid=%s&templateid=%s",
		s.UserID,
		s.Password,
		phone,
		s.Sender,
		url.QueryEscape(message),
		s.PEID,
		s.TemplateID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	log.Println("OTP sent successfully:", phone, "OTP =", otp)
	if m, ok := ctx.Value("otpMap").(map[string]string); ok {
		m[phone] = otp
	}

	return nil
}
