package domain

import "time"

type OTP struct {
	Phone     string    `bson:"phone"`
	Code      string    `bson:"code"`
	ExpiresAt time.Time `bson:"expiresAt"`
}
