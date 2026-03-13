package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserStatus string

const (
	USER_ACTIVE    UserStatus = "active"
	USER_DEACTIVE  UserStatus = "deactivated"
	USER_BLACKLIST UserStatus = "blacklisted"
	USER_DELETED   UserStatus = "delete"
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
	IsActive            UserStatus           `bson:"isActive" json:"isActive"`
	FcmToken            string               `bson:"fcmToken" json:"fcmToken"`
	AppStateStatus      string               `bson:"appStateStatus" json:"appStateStatus"`
	IsProfileComplete   bool                 `bson:"isProfileComplete" json:"isProfileComplete"`
	SelectedCity        string               `bson:"selectedCity" json:"selectedCity"`
	AmcPurchased        map[string]string    `bson:"amcPurchased" json:"amcPurchased"`
	ComplaintsSubmitted []primitive.ObjectID `bson:"complaintsSubmitted" json:"complaintsSubmitted"`
	ServiceOTP          string               `bson:"service_otp" json:"service_otp"`
	TotalExpense        float64              `bson:"totalExpense" json:"totalExpense"`
	Rating              string               `bson:"rating" json:"rating"`
	VehicleID           *primitive.ObjectID  `bson:"vehicleId,omitempty" json:"vehicleId,omitempty"`
	ComplaintsCount     int                  `bson:"complaintsCount" json:"complaintsCount"`
	IsNew               bool                 `bson:"isNew" json:"isNew"`
	PrimaryVehicleID    *primitive.ObjectID  `bson:"primaryVehicleId,omitempty" json:"primaryVehicleId,omitempty"`
	FallbackVehicleIDs  []primitive.ObjectID `bson:"fallbackVehicleIds,omitempty" json:"fallbackVehicleIds,omitempty"`
	CurrentAppVersion        string            `bson:"currentVerison" json:"currentVerison"`
	ForceUpdate         bool  				`bson:"forceUpdate" json:"forceUpdate"`
	ResponseMsg          string            `bson:"responseMsg" json:"responseMsg"`
	AndroidStoreURL    string               `bson:"androidStoreUrl" json:"androidStoreUrl"`
	IOSStoreURL        string 				`bson:"iosStoreUrl" json:"iosStoreUrl"`
}
