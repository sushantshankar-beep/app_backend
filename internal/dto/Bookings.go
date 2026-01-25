package dto

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type UserBookingDTO struct {
	ID            primitive.ObjectID `json:"id"`
	ServiceNumber string             `json:"serviceNumber"`
	Status        string             `json:"status"`
	FinalPrice    float64            `json:"finalPrice"`
	Issues        []string           `json:"issues,omitempty"`
	Name          string             `json:"name"`
	VehicleType   string             `json:"vehicleType"`
	CreatedAt     time.Time          `json:"createdAt"`
}

type UserBookingDetailDTO struct {
	ServiceNumber string            `json:"serviceNumber"`
	Status        string            `json:"status"`
	FinalPrice    float64           `json:"finalPrice"`
	VehicleNumber string            `json:"vehicleNumber"`
	Brand         string            `json:"brand"`
	Model         string            `json:"model"`
	ModelYear     int               `json:"modelYear"`
	FuelType      string            `json:"fuelType"`
	VehicleType   string            `json:"vehicleType"`
	Issues        []string          `json:"issues"`
	Timestamps    interface{}       `json:"timestamps"`
	UserName      string            `json:"userName"`
	CreatedAt     time.Time         `json:"createdAt"`
	Billing       BillingDetailsDTO `json:"billing"`
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
	ServiceNumber string             `json:"serviceNumber"`
	Status        string             `json:"status"`
	FinalPrice    float64            `json:"finalPrice"`
	VehicleNumber string             `json:"vehicleNumber"`
	Brand         string             `json:"brand"`
	Model         string             `json:"model"`
	ModelYear     int                `json:"modelYear"`
	VehicleType   string             `json:"vehicleType"`
	Issues        []string           `json:"issues,omitempty"`
	Name          string             `json:"name"`
	CreatedAt     time.Time          `json:"createdAt"`
}

type ProviderBookingDetailDTO struct {
	ServiceNumber string            `json:"serviceNumber"`
	Status        string            `json:"status"`
	FinalPrice    float64           `json:"finalPrice"`
	VehicleNumber string            `json:"vehicleNumber"`
	Brand         string            `json:"brand"`
	Model         string            `json:"model"`
	ModelYear     int               `json:"modelYear"`
	FuelType      string            `json:"fuelType"`
	VehicleType   string            `json:"vehicleType"`
	Issues        []string          `json:"issues"`
	Timestamps    interface{}       `json:"timestamps"`
	ProviderName  string            `json:"providerName"`
	CreatedAt     time.Time         `json:"createdAt"`
	Billing       BillingDetailsDTO `json:"billing"`
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
