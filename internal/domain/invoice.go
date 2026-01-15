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

	BillToName    string             `bson:"billToName" json:"billToName"`
	BillToAddress string             `bson:"billToAddress" json:"billToAddress"`

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
