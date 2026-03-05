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
   DEVICE TOKEN REPO PORT
============================================================ */

type DeviceTokenRepository interface {
	GetTokens(ctx context.Context, ownerID, ownerType string) ([]string, error)
	DeleteToken(ctx context.Context, token string) error
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

	app, err := firebase.NewApp(ctx, nil,
		option.WithCredentialsFile(credFile))
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

/* ============================================================
   PUBLIC METHODS
============================================================ */

func (f *FirebaseNotificationService) SendToProvider(
	ctx context.Context,
	providerID,
	serviceID,
	title,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, providerID, "provider", serviceID, title, body, data)
}

func (f *FirebaseNotificationService) SendToUser(
	ctx context.Context,
	userID,
	serviceID,
	title,
	body string,
	data map[string]string,
) error {

	return f.send(ctx, userID, "user", serviceID, title, body, data)
}

/* ============================================================
   CORE SEND
============================================================ */

func (f *FirebaseNotificationService) send(
	ctx context.Context,
	ownerID,
	ownerType,
	serviceID,
	title,
	body string,
	data map[string]string,
) error {

	// ---------- SAFE CONTEXT ----------
	ctx2 := ctx
	if ctx == nil || ctx.Err() != nil {
		ctx2 = context.Background()
	}

	ctx2, cancel := context.WithTimeout(ctx2, 10*time.Second)
	defer cancel()

	// ---------- PARSE IDS ----------
	ownerOID, err := primitive.ObjectIDFromHex(ownerID)
	if err != nil {
		return err
	}

	serviceOID := primitive.NilObjectID
	if serviceID != "" {
		serviceOID, err = primitive.ObjectIDFromHex(serviceID)
		if err != nil {
			return err
		}
	}

	// ---------- COPY PAYLOAD ----------
	payload := make(map[string]string)
	for k, v := range data {
		payload[k] = v
	}

	if serviceID != "" {
		payload["serviceId"] = serviceID
	}

	// ---------- SAVE NOTIFICATION ----------
	notif := &domain.Notification{
		OwnerID:   ownerOID,
		OwnerType: ownerType,
		ServiceID: serviceOID,
		Title:     title,
		Body:      body,
		Data:      payload,
		Read:      false,
		Status:    "sent",
		CreatedAt: time.Now(),
	}

	if err := f.notifRepo.Create(ctx2, notif); err != nil {
		log.Println("❌ notification save failed:", err)
	}

	// ---------- FETCH TOKENS ----------
	tokens, err := f.tokenRepo.GetTokens(ctx2, ownerID, ownerType)
	if err != nil {
		return err
	}

	if len(tokens) == 0 {
		log.Println("⚠️ no device tokens found")
		return nil
	}

	// ============================================================
	// ANDROID CONFIG (CUSTOM SOUND + HIGH PRIORITY)
	// ============================================================

	androidCfg := &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			Sound:        "custom",        // custom.mp3 inside res/raw
			ChannelID:    "high_priority", // must exist in app
			Priority:     messaging.PriorityHigh,
			DefaultSound: false,
		},
	}

	// ============================================================
	// IOS CONFIG (BACKGROUND + CUSTOM SOUND)
	// ============================================================

	apnsCfg := &messaging.APNSConfig{
		Headers: map[string]string{
			"apns-priority": "10",
		},
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Sound:            "custom.wav", // must exist in iOS bundle
			},
		},
	}

	// ============================================================
	// SINGLE TOKEN
	// ============================================================

	if len(tokens) == 1 {

		msg := &messaging.Message{
			Token: tokens[0],
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data:    payload,
			Android: androidCfg,
			APNS:    apnsCfg,
		}

		_, err := f.fcm.Send(ctx2, msg)
		return err
	}

	// ============================================================
	// MULTICAST
	// ============================================================

	mmsg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:    payload,
		Android: androidCfg,
		APNS:    apnsCfg,
	}

	resp, err := f.fcm.SendMulticast(ctx2, mmsg)
	if err != nil {
		return err
	}

	log.Printf(
		"📲 FCM multicast success=%d fail=%d",
		resp.SuccessCount,
		resp.FailureCount,
	)

	// ---------- CLEAN INVALID TOKENS ----------
	for i, r := range resp.Responses {
		if !r.Success {
			if messaging.IsUnregistered(r.Error) ||
				messaging.IsInvalidArgument(r.Error) {

				_ = f.tokenRepo.DeleteToken(ctx2, tokens[i])
			}
		}
	}

	return nil
}