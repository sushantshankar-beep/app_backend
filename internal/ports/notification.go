package ports

import "context"

type NotificationService interface {
	SendToProvider(
		ctx context.Context,
		providerID string,
		title string,
		body string,
		data map[string]string,
	) error
}
