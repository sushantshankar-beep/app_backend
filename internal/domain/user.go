package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserID string

type User struct {
	ID                  UserID               `bson:"_id,omitempty" json:"id"`
	UserCode            string               `bson:"userCode" json:"userCode"`
	Phone               string               `bson:"phone" json:"phone"`
	Name                string               `bson:"name" json:"name"`
	CreatedAt           time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
	Email               string               `bson:"email" json:"email"`
	ImageUrl            string               `bson:"image_url" json:"image_url"`
	ReferralCode        string               `bson:"referralCode" json:"referralCode"`
	IsActive            bool                 `bson:"isActive" json:"isActive"`
	FcmToken            string               `bson:"fcmToken" json:"fcmToken"`
	AppStateStatus      string               `bson:"appStateStatus" json:"appStateStatus"`
	IsProfileComplete   bool                 `bson:"isProfileComplete" json:"isProfileComplete"`
	SelectedCity        string               `bson:"selectedCity" json:"selectedCity"`
	AmcPurchased        map[string]string    `bson:"amcPurchased" json:"amcPurchased"`
	ComplaintsSubmitted []primitive.ObjectID `bson:"complaintsSubmitted" json:"complaintsSubmitted"`
	ServiceOTP          string               `bson:"service_otp" json:"service_otp"`
	VehicleID           *primitive.ObjectID  `bson:"vehicleId,omitempty" json:"vehicleId,omitempty"`
	PrimaryVehicleID    *primitive.ObjectID  `bson:"primaryVehicleId,omitempty" json:"primaryVehicleId,omitempty"`
	FallbackVehicleIDs  []primitive.ObjectID `bson:"fallbackVehicleIds,o1mitempty"`
}
