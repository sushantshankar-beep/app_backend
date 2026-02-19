package dto

import (
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type UserBookingDTO struct {
	ID            primitive.ObjectID `json:"id"`
	UserID        string             `json:"userId"`
	ServiceNumber string             `json:"serviceNumber"`
	ProfileURL    string      `json:"profileUrl"`
	ProviderProfileUrl    string      `json:"providerProfileUrl"`
	Status        string             `json:"status"`
	FinalPrice    float64            `json:"finalPrice"`
	Issues        []string           `json:"issues,omitempty"`
	UserName      string             `json:"userName"`
	ProviderName  string             `json:"providerName`
	VehicleType   string             `json:"vehicleType"`
	VehicleNumber string             `json:"vehicleNumber"`
	Brand         string             `json:"brand"`
	Model         string             `json:"model"`
	ModelYear     int                `json:"modelYear"`
	Rating               string      `json:"rating"`
	Cancelled     *domain.CancelInfo `json:"cancelled"`
	CancelledAt   *time.Time         `json:"cancelledAt"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
	ProviderRaisedComplaint bool                `json:"providerRaisedComplaint"`
	EtaMinutes    int64               `json:"etaMinutes"`
} 

type UserBookingDetailDTO struct {
	ID               primitive.ObjectID `json:"id"`
	UserID           string             `json:"userId"`
	ProviderID        domain.ProviderID             `json:"providerId"`
	ServiceNumber    string             `json:"serviceNumber"`
	ProfileURL    string      `json:"profileUrl"`
	ProviderProfileUrl    string      `json:"providerProfileUrl"`
	Status           string             `json:"status"`
	FinalPrice       float64            `json:"finalPrice"`
	VehicleNumber    string             `json:"vehicleNumber"`
	Brand            string             `json:"brand"`
	Model            string             `json:"model"`
	ModelYear        int                `json:"modelYear"`
	FuelType         string             `json:"fuelType"`
	VehicleType      string             `json:"vehicleType"`
	Issues           []string           `json:"issues"`
	Timestamps       interface{}        `json:"timestamps"`
	UserName         string             `json:"userName"`
	ProviderName     string             `json:"providerName"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	Billing          BillingDetailsDTO  `json:"billing"`
	UserLocation     UserLocation       `json:userLocation`
	ProviderLocation ProviderLocation   `json:providerLocation`
	Cancelled     *domain.CancelInfo `json:"cancelled"`
	CancelledAt   *time.Time         `json:"cancelledAt"`
	Complaint        *ComplaintDTO      `json:"complaint"`
	Rating               string             `json:"rating"`
	DistanceKm    float64              `json:"distanceKm"`
	EtaMinutes    int64               `json:"etaMinutes"`
}

type BillingDetailsDTO struct {
	ServiceCharge     float64 `json:"serviceCharge"`
	GSTPercent        float64 `json:"gstPercent"`
	GSTAmount         float64 `json:"gstAmount"`
	TotalPayable      float64 `json:"totalPayable"`
	PaymentStatus     string  `json:"paymentStatus"`
	CommissionPercent float64 `json:"commissionPercent,omitempty"`
	CommissionAmount  float64 `json:"commissionAmount,omitempty"`
	ProviderPayout    float64 `json:"providerPayout,omitempty"`
}

type ProviderBookingDTO struct {
	ID            primitive.ObjectID `json:"id"`
	ProviderID    string             `json:"providerId"`
	ProfileURL    string      `json:"profileUrl"`
	UserProfileUrl    string      `json:"userProfileUrl"`
	ServiceNumber string             `json:"serviceNumber"`
	Status        string             `json:"status"`
	FinalPrice    float64            `json:"finalPrice"`
	VehicleNumber string             `json:"vehicleNumber"`
	Brand         string             `json:"brand"`
	Model         string             `json:"model"`
	ModelYear     int                `json:"modelYear"`
	VehicleType   string             `json:"vehicleType"`
	Issues        []string           `json:"issues,omitempty"`
	ProviderName  string             `json:"providerName"`
	UserName      string             `json:"userName"`
	Rating               string      `json:"rating"`
	Cancelled     *domain.CancelInfo `json:"cancelled"`
	CancelledAt   *time.Time         `json:"cancelledAt"`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt     time.Time          `json:"updatedAt"`
	UserRaisedComplaint bool                `json:"userRaisedComplaint"`
}

type ProviderBookingDetailDTO struct {
	ID               primitive.ObjectID `json:"id"`
	ProviderID       string             `json:"providerId"`
	ProfileURL    string      `json:"profileUrl"`
	UserProfileUrl    string      `json:"userProfileUrl"`
	ServiceNumber    string             `json:"serviceNumber"`
	Status           string             `json:"status"`
	FinalPrice       float64            `json:"finalPrice"`
	VehicleNumber    string             `json:"vehicleNumber"`
	Brand            string             `json:"brand"`
	Model            string             `json:"model"`
	ModelYear        int                `json:"modelYear"`
	FuelType         string             `json:"fuelType"`
	VehicleType      string             `json:"vehicleType"`
	Issues           []string           `json:"issues"`
	Timestamps       interface{}        `json:"timestamps"`
	ProviderName     string             `json:"providerName"`
	UserName         string             `json:"userName"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt        time.Time          `json:"updatedAt"`
	Billing          BillingDetailsDTO  `json:"billing"`
	Rating               string      `json:"rating"`
	UserLocation     UserLocation       `json:userLocation`
	ProviderLocation ProviderLocation   `json:providerLocation`
	Cancelled     *domain.CancelInfo `json:"cancelled"`
	CancelledAt   *time.Time         `json:"cancelledAt"`
	Complaint        *ComplaintDTO      `json:"complaint"`
}

type ProviderBillingDetailsDTO struct {
	ServiceCharge float64 `json:"serviceCharge"`
	GSTPercent    float64 `json:"gstPercent"`
	GSTAmount     float64 `json:"gstAmount"`
	TotalPayable  float64 `json:"totalPayable"`
	PaymentStatus string  `json:"paymentStatus"`
}

type ProviderBookingResponse struct {
	Bookings []ProviderBookingDTO `json:"bookings"`
	Count    int                  `json:"count"`
}

type UserLocation struct {
	Lat  float64 `bson:"lat"`
	Long float64 `bson:"long"`
}

type ProviderLocation struct {
	Lat  float64 `bson:"lat"`
	Long float64 `bson:"long"`
}

type DashboardStats struct {
	AllTimeEarning    float64 `json:"allTimeEarning"`
	TodayEarning      float64 `json:"todayEarning"`
	ServicesCompleted int     `json:"servicesCompleted"`
	PaymentSettlement float64 `json:"paymentSettlement"`
	CancelledServices int     `json:"cancelledServices"`
}

type EarningDetail struct {
	ID          string             `json:"id"`
	ProviderId  primitive.ObjectID `json:"provider"`
	ServiceName string             `json:"serviceName"`
	Amount      float64            `json:"amount"`
	CreatedAt   string             `json:"createdAt"`
}

type EarningsResponse struct {
	Earnings []EarningDetail `json:"earnings"`
}

type TodayEarningsResponse struct {
	Total    float64         `json:"total"`
	Earnings []EarningDetail `json:"earnings"`
}

type ComplaintDTO struct {
	ID              string                `json:"_id"`
	ComplaintUserID string                `json:"complaintUserId"`
	ComplaintProviderID string             `json:"complaintProviderId"`
	ComplaintNumber string                `json:"complaintNumber"`
	Status          string                `json:"status"`
	ProviderIssue   *ProviderComplaintDTO `json:"providerComplaint"`
	UserIssue       *UserComplaintDTO     `json:"userComplaint"`
	Timeline        map[string]time.Time  `json:"timeline"`
	Remark          string                `json:remark`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type ProviderComplaintDTO struct {
	Problem  string    `json:"problem"`
	Photos   []string  `json:"photos"`
	RaisedAt time.Time `json:"raisedAt"`
}

type UserComplaintDTO struct {
	Problem  string    `json:"problem"`
	Photos   []string  `json:"photos"`
	RaisedAt time.Time `json:"raisedAt"`
}
