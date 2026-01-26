package service

import (
	"context"
	"log"
	"os"

	"app_backend/internal/ports"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
	"errors"
)

type FirebaseNotificationService struct {
	fcm *messaging.Client
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
		fcm: fcm,
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
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	log.Println("🔥 Firebase FCM initialized")

	return client, nil
}

func (f *FirebaseNotificationService) SendToProvider(
	ctx context.Context,
	providerID string,
	title string,
	body string,
	data map[string]string,
) error {

	tokens, err := f.tokenRepo.GetTokens(ctx, providerID)
	if err != nil {
	log.Printf("❌ FCM send error user=%s err=%v", userID, err)
	return err
}
	if err != nil || len(tokens) == 0 {
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

	resp, err := f.fcm.SendMulticast(ctx, msg)
	if err != nil {
		return err
	}

	log.Printf("📲 FCM provider=%s success=%d fail=%d",
		providerID,
		resp.SuccessCount,
		resp.FailureCount,
	)

	return nil
}

func (f *FirebaseNotificationService) SendToUser(
	ctx context.Context,
	userID string,
	title string,
	body string,
	data map[string]string,
) error {

	tokens, err := f.tokenRepo.GetTokens(ctx, userID)
	log.Printf("FCM user=%s tokens=%v err=%v", userID, tokens, err)
	if err != nil || len(tokens) == 0 {
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

	resp, err := f.fcm.SendMulticast(ctx, msg)
	if err != nil {
		log.Printf("❌ FCM send error user=%s err=%v", userID, err)
		return err
	}
	log.Printf("📲 FCM user=%s success=%d fail=%d",
		userID,
		resp.SuccessCount,
		resp.FailureCount,
	)
	
	return err
}
