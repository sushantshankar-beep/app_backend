package dto

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserBookingDTO struct {
	ID            primitive.ObjectID `json:"id"`
	UserID        string             `json:"userId"`
	ServiceNumber string             `json:"serviceNumber"`
	Status        string             `json:"status"`
	FinalPrice    float64            `json:"finalPrice"`
	Issues        []string           `json:"issues,omitempty"`
	UserName      string             `json:"userName"`
	ProviderName  string             `json:"providerName`
	VehicleType   string             `json:"vehicleType"`
	Ratings       string             `json:ratings`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
}

type UserBookingDetailDTO struct {
	ID               primitive.ObjectID `json:"id"`
	UserID           string             `json:"userId"`
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
	UserName         string             `json:"userName"`
	ProviderName     string             `json:"providerName"`
	CreatedAt        time.Time          `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
	Billing          BillingDetailsDTO  `json:"billing"`
	UserLocation     UserLocation       `json:userLocation`
	ProviderLocation ProviderLocation   `json:providerLocation`
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
	Ratings       string             `json:ratings`
	CreatedAt     time.Time          `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
}

type ProviderBookingDetailDTO struct {
	ID               primitive.ObjectID `json:"id"`
	ProviderID       string             `json:"providerId"`
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
	UpdatedAt       time.Time            `json:"updatedAt"`
	Billing          BillingDetailsDTO  `json:"billing"`
	UserLocation     UserLocation       `json:userLocation`
	ProviderLocation ProviderLocation   `json:providerLocation`
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
