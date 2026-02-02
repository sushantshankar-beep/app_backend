package dto

import "time"

type NotificationResponse struct {
	ID string `json:"id"`

	OwnerID   string `json:"ownerId"`
	OwnerType string `json:"ownerType"`

	ServiceID string `json:"serviceId"`

	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data"`

	Read   bool   `json:"read"`
	Status string `json:"status"`

	CreatedAt time.Time `json:"createdAt"`
}
