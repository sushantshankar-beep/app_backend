package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Invoice struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	InvoiceNumber string             `bson:"invoiceNumber" json:"invoiceNumber"`
	UserID        primitive.ObjectID `bson:"userId" json:"userId"`
	ServiceID     primitive.ObjectID `bson:"serviceId" json:"serviceId"`
	ProviderID    primitive.ObjectID `bson:"providerId" json:"providerId"`
	BillToName    string             `bson:"billToName" json:"billToName"`
	BillToAddress string             `bson:"billToAddress" json:"billToAddress"`
	BillToPhone   string             `bson:"billToPhone" json:"billToPhone"`
	ProviderName  string             `bson:"providerName" json:"providerName"`
	ProviderAddress string           `bson:"providerAddress" json:"providerAddress"`
	ProviderPhone string             `bson:"providerPhone" json:"providerPhone"`
	ProviderGST   string             `bson:"providerGst" json:"providerGst"`
	VehicleBrand  string             `bson:"vehicleBrand" json:"vehicleBrand"`
	VehicleModel  string             `bson:"vehicleModel" json:"vehicleModel"`
	VehicleNumber string             `bson:"vehicleNumber" json:"vehicleNumber"`
	VehicleYear   int                `bson:"vehicleYear" json:"vehicleYear"`
	VehicleType   string             `bson:"vehicleType" json:"vehicleType"`
	FuelType      string             `bson:"fuelType" json:"fuelType"`
	ServiceType   string             `bson:"serviceType" json:"serviceType"`
	ServiceDate   *time.Time         `bson:"serviceDate" json:"serviceDate"`
	Items         []InvoiceItem      `bson:"items" json:"items"`
	SubTotal      float64            `bson:"subTotal" json:"subTotal"`
	TaxAmount     float64            `bson:"taxAmount" json:"taxAmount"`
	TotalAmount   float64            `bson:"totalAmount" json:"totalAmount"`
	PDFUrl        string             `bson:"pdfUrl" json:"pdfUrl"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
}

type InvoiceItem struct {
	Name        string  `bson:"name" json:"name"`
	Qty         int     `bson:"qty" json:"qty"`
	Price       float64 `bson:"price" json:"price"`
	GSTPercent  float64 `bson:"gstPercent" json:"gstPercent"`
	GrossAmount float64 `bson:"grossAmount" json:"grossAmount"`
}