package domain

import "time"

type UserID string

type User struct {
	ID        UserID    `bson:"_id,omitempty" json:"id"`
	Phone     string    `bson:"phone" json:"phone"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
