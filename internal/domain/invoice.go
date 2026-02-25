package domain

import (

	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Invoice struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	InvoiceNumber  string             `bson:"invoiceNumber" json:"invoiceNumber"`
	InvoiceDate    time.Time          `bson:"invoiceDate" json:"invoiceDate"`
	ServiceDate    *time.Time         `bson:"serviceDate,omitempty" json:"serviceDate,omitempty"`
	UserID         primitive.ObjectID `bson:"userId" json:"userId"`
	ServiceID      string             `bson:"serviceId" json:"serviceId"`
	ServiceNumber  string              `bson:"serviceNumber" json:"serviceNumber"`
	CompanyInfo    CompanyInfo        `bson:"companyInfo" json:"companyInfo"`
	CustomerInfo   CustomerInfo       `bson:"customer" json:"customer"`
	VehicleDetails VehicleInfo        `bson:"vehicle" json:"vehicle"`
	ServiceInfo    ServiceInfo        `bson:"service" json:"service"`
	PricingDeatils PricingInfo        `bson:"pricing" json:"pricing"`
	Transaction    InvoiceTransaction         `bson:"transaction" json:"transaction"`
	PDFUrl         string             `bson:"pdfUrl" json:"pdfUrl"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}

type CustomerInfo struct {
	Name  string `bson:"name" json:"name"`
	Phone string `bson:"phone" json:"phone"`
	Email string `bson:"email" json:"email"`
}

type CompanyInfo struct {
	Name    string
	Address string
	GST     string
	Phone   string
	Email   string
}
type VehicleInfo struct {
	Brand         string `bson:"brand" json:"brand"`
	Model         string `bson:"model" json:"model"`
	Year          int    `bson:"year" json:"year"`
	VehicleType   string `bson:"vehicleType" json:"vehicleType"`
	VehicleNumber string `bson:"vehicleNumber" json:"vehicleNumber"`
	FuelType      string `bson:"fuelType" json:"fuelType"`
}

type ServiceInfo struct {
	Type          string        `bson:"type" json:"type"`
	Status        ServiceStatus `bson:"status" json:"status"`
	PaymentStatus PaymentStatus `bson:"paymentStatus" json:"paymentStatus"`
}

type PricingInfo struct {
    ServiceCharge       float64            `bson:"serviceCharge" json:"serviceCharge"`
    TotalDiscount       float64            `bson:"totalDiscount,omitempty" json:"totalDiscount,omitempty"`
    AmountAfterDiscount float64            `bson:"amountAfterDiscount,omitempty" json:"amountAfterDiscount,omitempty"`
    GSTPercent          float64            `bson:"gstPercent,omitempty" json:"gstPercent,omitempty"`
    GST                 float64            `bson:"gst" json:"gst"`
    Total               float64            `bson:"total" json:"total"`
    AppliedPromo        *AppliedPromoSummary `bson:"appliedPromo" json:"appliedPromo"`
    AppliedDiscount     *AppliedDiscountSummary   `bson:"appliedDiscount" json:"appliedDiscount"`
}

type InvoiceTransaction struct {
	PaymentMode        string    `bson:"method,omitempty"`
}