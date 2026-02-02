package service

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/repository"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/api/option"
)

/* ============================================================
   SERVICE
============================================================ */

type FirebaseNotificationService struct {
	fcm       *messaging.Client
	tokenRepo DeviceTokenRepository
	notifRepo *repository.NotificationRepo
}

/* ============================================================
   ERRORS
============================================================ */

var ErrMissingFirebaseCred = errors.New(
	"FIREBASE_CREDENTIALS_JSON environment variable not set",
)

/* ============================================================
   DEVICE TOKEN REPO PORT  ✅ FIXED
============================================================ */

type DeviceTokenRepository interface {
	GetTokens(
		ctx context.Context,
		ownerID string,
		ownerType string,
	) ([]string, error)
}

/* ============================================================
   CONSTRUCTOR
============================================================ */

func NewFirebaseNotificationService(
	fcm *messaging.Client,
	tokenRepo DeviceTokenRepository,
	notifRepo *repository.NotificationRepo,
) ports.NotificationService {

	return &FirebaseNotificationService{
		fcm:       fcm,
		tokenRepo: tokenRepo,
		notifRepo: notifRepo,
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

	return f.send(ctx, providerID, "provider", title, body, data)
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

	return f.send(ctx, userID, "user", title, body, data)
}

/* ============================================================
   CORE SEND + SAVE
============================================================ */

func (f *FirebaseNotificationService) send(
	ctx context.Context,
	ownerID string,
	ownerType string,
	title string,
	body string,
	data map[string]string,
) error {

	// never use cancelled ctx
	ctx2 := ctx
	if ctx == nil || ctx.Err() != nil {
		ctx2 = context.Background()
	}

	ctx2, cancel := context.WithTimeout(ctx2, 6*time.Second)
	defer cancel()

	// ------------------------------
	// 🔎 PARSE OWNER ID
	// ------------------------------
	ownerOID, err := primitive.ObjectIDFromHex(ownerID)
	if err != nil {
		return err
	}

	// ------------------------------
	// 📦 SERVICE ID (optional)
	// ------------------------------
	var serviceOID primitive.ObjectID
	if sid, ok := data["serviceId"]; ok && sid != "" {
		serviceOID, _ = primitive.ObjectIDFromHex(sid)
	}

	// ------------------------------
	// 💾 SAVE TO DB FIRST
	// ------------------------------
	notif := &domain.Notification{
		OwnerID:   ownerOID,
		OwnerType: ownerType,
		ServiceID: serviceOID,

		Title: title,
		Body:  body,
		Data:  data,

		Read:   false,
		Status: "sent",

		CreatedAt: time.Now(),
	}

	if err := f.notifRepo.Create(ctx2, notif); err != nil {
		log.Println("❌ Notification DB insert failed:", err)
	}

	// ------------------------------
	// 🔎 FETCH TOKENS  ✅ FIXED
	// ------------------------------
	tokens, err := f.tokenRepo.GetTokens(ctx2, ownerID, ownerType)
	if err != nil {
		log.Printf("❌ FCM token lookup failed owner=%s err=%v",
			ownerID, err)
		return err
	}

	if len(tokens) == 0 {
		log.Printf("⚠️ No FCM tokens owner=%s", ownerID)
		return nil
	}

	// ------------------------------
	// ✅ SINGLE TOKEN
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
	// ✅ MULTI TOKEN
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
