package service

import (
	"context"
	"log"
)

type FirebaseNotificationService struct {
}

func NewFirebaseNotificationService() *FirebaseNotificationService {
	return &FirebaseNotificationService{}
}

func (f *FirebaseNotificationService) SendToProvider(
	ctx context.Context,
	providerID string,
	title string,
	body string,
	data map[string]string,
) error {

	// TODO: Plug Firebase Admin SDK here
	log.Printf(
		"🔔 Push → provider=%s | %s - %s | data=%v\n",
		providerID,
		title,
		body,
		data,
	)

	return nil
}
