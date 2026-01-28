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

/* ============================================================
   SERVICE
============================================================ */

type FirebaseNotificationService struct {
	fcm       *messaging.Client
	tokenRepo DeviceTokenRepository
}

/* ============================================================
   ERRORS
============================================================ */

var ErrMissingFirebaseCred = errors.New(
	"FIREBASE_CREDENTIALS_JSON environment variable not set",
)

/* ============================================================
   DEVICE TOKEN REPO PORT
============================================================ */

type DeviceTokenRepository interface {
	GetTokens(ctx context.Context, ownerID string) ([]string, error)
}

/* ============================================================
   CONSTRUCTOR
============================================================ */

func NewFirebaseNotificationService(
	fcm *messaging.Client,
	tokenRepo DeviceTokenRepository,
) ports.NotificationService {

	return &FirebaseNotificationService{
		fcm:       fcm,
		tokenRepo: tokenRepo,
	}
}

/* ============================================================
   INIT FIREBASE CLIENT
============================================================ */

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

/* ============================================================
   PROVIDER
============================================================ */

func (f *FirebaseNotificationService) SendToProvider(
	ctx context.Context,
	providerID string,
	title string,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, providerID, title, body, data)
}

/* ============================================================
   USER
============================================================ */

func (f *FirebaseNotificationService) SendToUser(
	ctx context.Context,
	userID string,
	title string,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, userID, title, body, data)
}

/* ============================================================
   CORE SEND (FIXED)
============================================================ */

func (f *FirebaseNotificationService) send(
	ctx context.Context,
	ownerID string,
	title string,
	body string,
	data map[string]string,
) error {

	// never use cancelled ctx
	ctx2 := ctx
	if ctx == nil || ctx.Err() != nil {
		ctx2 = context.Background()
	}

	// timeout for mongo + FCM
	ctx2, cancel := context.WithTimeout(ctx2, 6*time.Second)
	defer cancel()

	// ------------------------------
	// 🔎 FETCH TOKENS
	// ------------------------------
	tokens, err := f.tokenRepo.GetTokens(ctx2, ownerID)
	if err != nil {
		log.Printf("❌ FCM token lookup failed owner=%s err=%v",
			ownerID, err)
		return err
	}

	log.Println("📲 FCM TOKENS =", tokens)

	if len(tokens) == 0 {
		log.Printf("⚠️ No FCM tokens owner=%s", ownerID)
		return nil
	}

	// ------------------------------
	// ✅ SINGLE TOKEN → Send()
	// ------------------------------
	if len(tokens) == 1 {

		msg := &messaging.Message{
			Token: tokens[0],
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: data,
		}

		resp, err := f.fcm.Send(ctx2, msg)

		log.Println("🔥 FCM SEND SINGLE RESPONSE =", resp)
		log.Println("🔥 FCM SEND SINGLE ERROR =", err)

		return err
	}

	// ------------------------------
	// ✅ MULTI TOKEN → SendMulticast()
	// ------------------------------
	mmsg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	resp, err := f.fcm.SendMulticast(ctx2, mmsg)

	if err != nil {
		log.Println("❌ FCM MULTICAST ERROR:", err)
		return err
	}

	log.Printf(
		"📲 FCM MULTI owner=%s success=%d fail=%d",
		ownerID,
		resp.SuccessCount,
		resp.FailureCount,
	)

	for i, r := range resp.Responses {
		if !r.Success {
			log.Printf("❌ token=%s err=%v",
				tokens[i],
				r.Error,
			)
		}
	}

	return nil
}

/* ============================================================
   TEST FUNCTION (OPTIONAL)
============================================================ */

func TestFCMSend(client *messaging.Client) {

	ctx := context.Background()

	token := "PASTE_REAL_TOKEN"

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
