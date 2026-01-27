package dto

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InvoiceData struct {
	InvoiceNumber string    `json:"invoiceNumber"`
	InvoiceDate   string    `json:"invoiceDate"`
	ServiceDate   *string   `json:"serviceDate"`
	Provider      Provider  `json:"provider"`
	Customer      Customer  `json:"customer"`
	Vehicle       Vehicle   `json:"vehicle"`
	Service       Service   `json:"service"`
	Pricing       Pricing   `json:"pricing"`
}

type Provider struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	GSTNumber string `json:"gstNumber"`
	Phone     string `json:"phone"`
}

type Customer struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

type Vehicle struct {
	Brand         string `json:"brand"`
	Model         string `json:"model"`
	Year          int    `json:"year"`
	VehicleType   string `json:"vehicleType"`
	VehicleNumber string `json:"vehicleNumber"`
	FuelType      string `json:"fuelType"`
}

type Service struct {
	Type          string   `json:"type"`
	Problems      []string `json:"problems"`
	Status        string   `json:"status"`
	PaymentStatus string   `json:"paymentStatus"`
}

type Pricing struct {
	ServiceCharge string `json:"serviceCharge"`
	Discount      string `json:"discount"`
	Subtotal      string `json:"subtotal"`
	GST           string `json:"gst"`
	Total         string `json:"total"`
}

type AcceptedService struct {
	ID             primitive.ObjectID `bson:"_id"`
	User           UserInfo           `bson:"user"`
	Provider       *ProviderInfo      `bson:"provider"`
	ServiceRequest ServiceRequestInfo `bson:"serviceRequest"`
	Status         string             `bson:"status"`
	PaymentStatus  string             `bson:"paymentStatus"`
	FinalPrice     float64            `bson:"finalPrice"`
	CompletedAt    *time.Time         `bson:"completedAt"`
}

type UserInfo struct {
	Name    string `bson:"name"`
	Phone   string `bson:"phone"`
	Address string `bson:"address"`
}

type ProviderInfo struct {
	Name        string `bson:"name"`
	CompanyName string `bson:"companyName"`
	Address     string `bson:"address"`
	GSTNumber   string `bson:"gstNumber"`
	Phone       string `bson:"phone"`
}

type ServiceRequestInfo struct {
	Brand         string   `bson:"brand"`
	Model         string   `bson:"model"`
	Year          int      `bson:"year"`
	VehicleType   string   `bson:"vehicleType"`
	VehicleNumber string   `bson:"vehicleNumber"`
	FuelType      string   `bson:"fuelType"`
	ServiceType   string   `bson:"serviceType"`
	Problems      []string `bson:"problems"`
}


