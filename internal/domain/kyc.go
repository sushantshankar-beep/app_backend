package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KYCStatus string

const (
	KYC_PENDING  KYCStatus = "PENDING"
	KYC_APPROVED KYCStatus = "APPROVED"
	KYC_REJECTED KYCStatus = "REJECTED"
)

type ProviderKYC struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	ProviderID primitive.ObjectID `bson:"providerId" json:"providerId"`

	DocumentType string `bson:"documentType" json:"documentType"`

	DocumentURL        string `bson:"documentUrl" json:"documentUrl"`
	ElectricityBillURL string `bson:"electricityBillUrl" json:"electricityBillUrl"`
	CancelledChequeURL string `bson:"cancelledChequeUrl" json:"cancelledChequeUrl"`

	AccountHolderName string `bson:"accountHolderName" json:"accountHolderName"`
	AccountNumber     string `bson:"accountNumber" json:"accountNumber"`
	BranchName        string `bson:"branchName" json:"branchName"`
	IFSC              string `bson:"ifsc" json:"ifsc"`
	UPIID             string `bson:"upiId,omitempty" json:"upiId,omitempty"`
	GSTNumber         string `bson:"gstNumber,omitempty" json:"gstNumber,omitempty"`

	Status KYCStatus `bson:"status" json:"status"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}
