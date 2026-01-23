package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type KYCStatus string

const (
	KYC_PENDING  KYCStatus = "PENDING"
	KYC_APPROVED KYCStatus = "APPROVED"
	KYC_REJECTED KYCStatus = "REJECTED"
)

type DocumentType string

const (
	DOC_AADHAAR_FRONT DocumentType = "AADHAAR_FRONT"
	DOC_AADHAAR_BACK  DocumentType = "AADHAAR_BACK"
	DOC_PAN           DocumentType = "PAN"
)

type VerificationStatus string

const (
	VERIFICATION_PENDING  VerificationStatus = "PENDING"
	VERIFICATION_APPROVED VerificationStatus = "APPROVED"
	VERIFICATION_REJECTED VerificationStatus = "REJECTED"
)

type KYCDocument struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type     DocumentType       `bson:"type" json:"type"`
	URL      string             `bson:"url" json:"url"`
	Verified VerificationStatus `bson:"verified" json:"verified"`
}

type ProviderBankDetails struct {
	AccountHolderName string `bson:"accountHolderName" json:"accountHolderName"`
	AccountNumber     string `bson:"accountNumber" json:"accountNumber"`
	IFSC              string  `bson:"ifsc" json:"ifsc"`
	BranchName        string `bson:"branchName" json:"branchName"`
	UPIID             string `bson:"upiId,omitempty" json:"upiId,omitempty"`
	GSTNumber          string    `bson:"gstNumber,omitempty" json:"gstNumber,omitempty"`
}

type ProviderKYC struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ProviderID primitive.ObjectID  `bson:"providerId" json:"providerId"`
	Documents  []KYCDocument       `bson:"documents" json:"documents"`
	Bank       ProviderBankDetails `bson:"bank" json:"bank"`
	Status     KYCStatus           `bson:"status" json:"status"`
	CreatedAt  time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time           `bson:"updatedAt" json:"updatedAt"`
}
