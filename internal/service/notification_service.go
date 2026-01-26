package service

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"app_backend/internal/ports"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type FirebaseNotificationService struct {
	fcm       *messaging.Client
	tokenRepo DeviceTokenRepository
}

var ErrMissingFirebaseCred = errors.New(
	"FIREBASE_CREDENTIALS_JSON environment variable not set",
)

type DeviceTokenRepository interface {
	GetTokens(ctx context.Context, ownerID string) ([]string, error)
}

func NewFirebaseNotificationService(
	fcm *messaging.Client,
	tokenRepo DeviceTokenRepository,
) ports.NotificationService {

	return &FirebaseNotificationService{
		fcm:       fcm,
		tokenRepo: tokenRepo,
	}
}

func InitFirebaseClient() (*messaging.Client, error) {
	ctx := context.Background()

	credFile := os.Getenv("FIREBASE_CREDENTIALS_JSON")
	if credFile == "" {
		return nil, ErrMissingFirebaseCred
	}

	app, err := firebase.NewApp(
		ctx,
		&firebase.Config{
			ProjectID: "vahanwire-d8ece",
		},
		option.WithCredentialsFile(credFile),
	)
	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	log.Println("🔥 Firebase FCM initialized")

	return client, nil
}

/* ================= PROVIDER ================= */

func (f *FirebaseNotificationService) SendToProvider(
	ctx context.Context,
	providerID string,
	title string,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, providerID, title, body, data)
}

/* ================= USER ================= */

func (f *FirebaseNotificationService) SendToUser(
	ctx context.Context,
	userID string,
	title string,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, userID, title, body, data)
}

/* ================= CORE SEND ================= */

func (f *FirebaseNotificationService) send(
	ctx context.Context,
	ownerID string,
	title string,
	body string,
	data map[string]string,
) error {

	// never allow cancelled ctx
	ctx2 := ctx
	if ctx == nil || ctx.Err() != nil {
		ctx2 = context.Background()
	}

	// timeout for mongo + FCM
	ctx2, cancel := context.WithTimeout(ctx2, 6*time.Second)
	defer cancel()

	tokens, err := f.tokenRepo.GetTokens(ctx2, ownerID)
	if err != nil {
		log.Printf("❌ FCM token lookup failed owner=%s err=%v", ownerID, err)
		return err
	}

	if len(tokens) == 0 {
		log.Printf("⚠️ No FCM tokens owner=%s", ownerID)
		return nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := f.fcm.SendMulticast(ctx2, msg)
	if err != nil {
		log.Printf("❌ FCM send failed owner=%s err=%v", ownerID, err)
		return err
	}

	log.Printf(
		"📲 FCM owner=%s success=%d fail=%d",
		ownerID,
		resp.SuccessCount,
		resp.FailureCount,
	)

	return nil
}
