package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AcceptedServiceID string

type AcceptedService struct {
	ID                   primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	ServiceRequest       primitive.ObjectID   `bson:"serviceRequest" json:"serviceRequest"`
	ServiceRequestID     int64                `bson:"serviceRequestId" json:"serviceRequestId"`
	NumericID            int64                `bson:"id" json:"numericId"`
	User                 primitive.ObjectID   `bson:"user" json:"user"`
	NotToSendProviders   []primitive.ObjectID `bson:"notToSendProviders,omitempty" json:"notToSendProviders,omitempty"`
	Provider             primitive.ObjectID   `bson:"provider" json:"provider"`
	AcceptedBid          primitive.ObjectID   `bson:"acceptedBid" json:"acceptedBid"`
	OTP                  OTPInfo              `bson:"otp" json:"otp"`
	Status               string               `bson:"status" json:"status"`
	ReachedAt            *time.Time           `bson:"reachedAt,omitempty" json:"reachedAt,omitempty"`
	StartedAt            *time.Time           `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt          *time.Time           `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CancelledAt          *time.Time           `bson:"cancelledAt,omitempty" json:"cancelledAt,omitempty"`
	ExpiresAt            *time.Time           `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	OTPVerifiedAt        *time.Time           `bson:"otpVerifiedAt,omitempty" json:"otpVerifiedAt,omitempty"`
	JobStartedAt         *time.Time           `bson:"jobStartedAt,omitempty" json:"jobStartedAt,omitempty"`
	CancelledBy          string               `bson:"cancelledBy,omitempty" json:"cancelledBy,omitempty"`
	BasePrice            float64              `bson:"basePrice" json:"basePrice"`
	FinalPrice           float64              `bson:"finalPrice" json:"finalPrice"`
	PaymentStatus        string               `bson:"paymentStatus" json:"paymentStatus"`
	ComplaintUser        *primitive.ObjectID  `bson:"complaintUser,omitempty" json:"complaintUser,omitempty"`
	ComplaintProvider    *primitive.ObjectID  `bson:"complaintProvider,omitempty" json:"complaintProvider,omitempty"`
	OrderID              string               `bson:"orderId,omitempty" json:"orderId,omitempty"`
	ServiceType          string               `bson:"serviceType,omitempty" json:"serviceType,omitempty"`
	Issues               []string             `bson:"issues,omitempty" json:"issues,omitempty"`
	ProviderLocation     *Location            `bson:"providerLocation,omitempty" json:"providerLocation,omitempty"`
	ServiceLocation      *Location            `bson:"serviceLocation,omitempty" json:"serviceLocation,omitempty"`
	CreatedAt            time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type OTPInfo struct {
	Code       string     `bson:"code,omitempty" json:"code,omitempty"`
	Verified   bool       `bson:"verified" json:"verified"`
	VerifiedAt *time.Time `bson:"verifiedAt,omitempty" json:"verifiedAt,omitempty"`
}

type DashboardStats struct {
	AllTimeEarning    float64 `json:"allTimeEarning"`
	TodaysEarning     float64 `json:"todaysEarning"`
	ServicesCompleted int     `json:"servicesCompleted"`
	CancelledServices int     `json:"cancelledServices"`
}