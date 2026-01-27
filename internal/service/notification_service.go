package service

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"app_backend/internal/ports"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
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

	opt := option.WithCredentialsFile(credFile)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, err
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	log.Println("🔥 Firebase FCM initialized (GO)")

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
func TestFCMSend(client *messaging.Client) {

	ctx := context.Background()

	token := "eiVvN1XiSWy4HB1dlLsp32:APA91bGdZPNncTcGYF4sT7r0N7KDQfyX8vbuI5o2t847z0Aaf756sA7VNjerh23LQM4jHWIBwuxDZ4akjc2l5FLTmPVxM9CQyQPGGo4N1LxCI4DXN5XwnUY"

	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: "🔥 Test from Go backend",
			Body:  "If you see this, FCM works!",
		},
		Data: map[string]string{
			"type": "test",
		},
	}

	resp, err := client.Send(ctx, msg)

	log.Println("FCM TEST RESPONSE =", resp)
	log.Println("FCM TEST ERROR =", err)
}

