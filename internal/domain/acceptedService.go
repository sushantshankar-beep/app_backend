package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AcceptedServiceID string

type AcceptedService struct {
	ID                    primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	ServiceRequest        primitive.ObjectID   `bson:"serviceRequest" json:"serviceRequest"`
	ServiceNumber         string               `bson:"serviceNumber" json:"serviceNumber"`
	NumericID             int64                `bson:"id" json:"numericId"`
	User                  primitive.ObjectID   `bson:"user" json:"user"`
	NotToSendProviders    []primitive.ObjectID `bson:"notToSendProviders,omitempty" json:"notToSendProviders,omitempty"`
	Provider              primitive.ObjectID   `bson:"provider" json:"provider"`
	AcceptedBid           primitive.ObjectID   `bson:"acceptedBid" json:"acceptedBid"`
	OTP                   OTPInfo              `bson:"otp" json:"otp"`
	Status                ServiceStatus        `bson:"status" json:"status"`
	ReachedAt             *time.Time           `bson:"reachedAt,omitempty" json:"reachedAt,omitempty"`
	StartedAt             *time.Time           `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt           *time.Time           `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	CancelledAt           *time.Time           `bson:"cancelledAt,omitempty" json:"cancelledAt,omitempty"`
	ExpiresAt             *time.Time           `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	OTPVerifiedAt         *time.Time           `bson:"otpVerifiedAt,omitempty" json:"otpVerifiedAt,omitempty"`
	JobStartedAt          *time.Time           `bson:"jobStartedAt,omitempty" json:"jobStartedAt,omitempty"`
	CancelledBy           string               `bson:"cancelledBy,omitempty" json:"cancelledBy,omitempty"`
	BasePrice             float64              `bson:"basePrice" json:"basePrice"`
	FinalPrice            float64              `bson:"finalPrice" json:"finalPrice"`
	PaymentStatus         PaymentStatus        `bson:"paymentStatus" json:"paymentStatus"`
	ComplaintUser         *primitive.ObjectID  `bson:"complaintUser,omitempty" json:"complaintUser,omitempty"`
	ComplaintProvider     *primitive.ObjectID  `bson:"complaintProvider,omitempty" json:"complaintProvider,omitempty"`
	OrderID               string               `bson:"orderId,omitempty" json:"orderId,omitempty"`
	ServiceType           string               `bson:"serviceType,omitempty" json:"serviceType,omitempty"`
	Issues                []string             `bson:"issues,omitempty" json:"issues,omitempty"`
	ProviderLocation      *ProviderLocation    `bson:"providerLocation,omitempty" json:"providerLocation,omitempty"`
	RetryCount            int                  `bson:"retryCount" json:"retryCount"`
	MaxRetries            int                  `bson:"maxRetries" json:"maxRetries"`
	CreatedAt             time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt             time.Time            `bson:"updatedAt" json:"updatedAt"`
	FuelType              string               `bson:"fuelType" json:"fuelType"`
	VehicleType           string               `bson:"vehicleType" json:"vehicleType"`
	VehicleNumber         string               `bson:"vehicleNumber" json:"vehicleNumber"`
	Brand                 string               `bson:"brand" json:"brand"`
	ModelYear             int                  `bson:"modelYear" json:"modelYear"`
	Model                 string               `bson:"model" json:"model"`
	Timestamps            *ServiceTimestamps   `bson:"timestamps,omitempty"`
	Cancelled             *CancelInfo          `bson:"cancelled,omitempty"`
	UserLocation          *UserLocation        `bson:"userLocation" json:"userLocation"`
	CancelledByProvider   bool                 `bson:"cancelledByProvider" json:"cancelledByProvider"`
	CancelledProviderID   string               `bson:"cancelledProviderID" json:"cancelledProviderID"`
	PendingCoupon   *CouponApplyResult      `bson:"pendingCoupon,omitempty"   json:"pendingCoupon,omitempty"`
	AmountPaidByUser float64                   `bson:"amountPaidByUser" json:"amountPaidByUser"`
    TotalDiscount    float64                   `bson:"totalDiscount" json:"totalDiscount"`
    AppliedPromo     *AppliedPromoSummary      `bson:"appliedPromo,omitempty" json:"appliedPromo,omitempty"`
    AppliedDiscount  *AppliedDiscountSummary   `bson:"appliedDiscount,omitempty" json:"appliedDiscount,omitempty"`
}
type UserLocation struct {
	Lat  float64 `bson:"lat"`
	Long float64 `bson:"long"`
}

type ProviderLocation struct {
	Lat  float64 `bson:"lat"`
	Long float64 `bson:"long"`
}

type ServiceTimestamps struct {
	CreatedAt    *time.Time `bson:"createdAt,omitempty"`
	StartedAt    *time.Time `bson:"startedAt,omitempty"`
	ReachedAt    *time.Time `bson:"reachedAt,omitempty"`
	InProgressAt *time.Time `bson:"inProgressAt,omitempty"`
	CompletedAt  *time.Time `bson:"completedAt,omitempty"`
	CancelledAt  *time.Time `bson:"cancelledAt,omitempty"`
}

type CancelInfo struct {
	By     string `bson:"by"` // "user" | "provider"
	Reason string `bson:"reason"`
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

func CanTransition(from, to ServiceStatus) bool {
	allowed := map[ServiceStatus][]ServiceStatus{

		// Pre-runtime → Start Job
		StatusCreated:          {StatusSearching, StatusConfirmed, StatusCancelled},
		StatusSearching:        {StatusProviderAssigned, StatusConfirmed, StatusStarted},
		StatusProviderAssigned: {StatusConfirmed, StatusNotStarted, StatusStarted},
		StatusAssigned:         {StatusConfirmed, StatusStarted, StatusCancelled},
		StatusConfirmed:        {StatusStarted, StatusCancelled},
		StatusNotStarted:       {StatusStarted, StatusCancelled},

		// Runtime journey
		StatusStarted:         {StatusReachedLocation},
		StatusReachedLocation: {StatusOTPVerified},

		// OTP → Service
		StatusOTPVerified: {StatusInProgress},

		StatusInProgress: {StatusCompleted},

		// End states
		StatusCompleted: {},
		StatusCancelled: {},
	}

	if from == to {
		return true
	}

	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}

type ServiceStatus string

const (
	StatusCreated          ServiceStatus = "created"
	StatusAssigned         ServiceStatus = "assigned"
	StatusStarted          ServiceStatus = "started"
	StatusCompleted        ServiceStatus = "completed"
	StatusCancelled        ServiceStatus = "cancelled"
	StatusSearching        ServiceStatus = "searching"
	StatusProviderAssigned ServiceStatus = "provider_assigned"
	StatusNotStarted       ServiceStatus = "not_started"
	StatusReachedLocation  ServiceStatus = "reached_location"
	StatusOTPVerified      ServiceStatus = "otp_verified"

	StatusInProgress ServiceStatus = "in_progress"
	StatusConfirmed  ServiceStatus = "confirmed"
)
