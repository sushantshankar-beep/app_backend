package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserID string

type User struct {
	ID                  UserID               `bson:"_id,omitempty" json:"id"`
	Phone               string               `bson:"phone" json:"phone"`
	Name                string 				 `bson:"name" json:"name"`
	CreatedAt           time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
	Email               string               `bson:"email" json:"email"`
	ReferralCode        string               `bson:"referralCode" json:"referralCode"`
	IsActive            bool                 `bson:"isActive" json:"isActive"`
	FcmToken            string               `bson:"fcmToken" json:"fcmToken"`
	AppStateStatus      string               `bson:"appStateStatus" json:"appStateStatus"`
	SelectedCity        string               `bson:"selectedCity" json:"selectedCity"`
	AmcPurchased        map[string]string    `bson:"amcPurchased" json:"amcPurchased"`
	ComplaintsSubmitted []primitive.ObjectID `bson:"complaintsSubmitted" json:"complaintsSubmitted"`
	ServiceOTP         string 				 `bson:"service_otp" json:"service_otp"`
}
