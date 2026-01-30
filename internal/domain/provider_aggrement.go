package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Paragraph struct {
	Number  string `bson:"number" json:"number"`
	Title   string `bson:"title" json:"title"`
	Content string `bson:"content" json:"content"`
}

type Agreement struct {
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Title             string             `bson:"title" json:"title"`
	Paragraphs        []Paragraph        `bson:"paragraphs" json:"paragraphs"`
	AgreementOf       string             `bson:"agreementOf" json:"agreementOf"`
	AgreementFor      string             `bson:"agreementFor" json:"agreementFor"`
	CompanyName       string             `bson:"companyName" json:"companyName"`
	Annexures         string             `bson:"annexures" json:"annexures"`
	CommercialTerms   string             `bson:"commercialTerms" json:"commercialTerms"`
	MarketplaceFee    string             `bson:"marketplaceFee" json:"marketplaceFee"`
	PaymentGateway    string             `bson:"paymentGateway" json:"paymentGateway"`
	AdditionalNotes   string             `bson:"additionalNotes" json:"additionalNotes"`
	DateTime          string             `bson:"dateTime,omitempty"`
	Place             string             `bson:"place,omitempty"`
	CommissionPercent string             `bson:"commissionPercent" json:"commissionPercent"`
	ProviderName       string             `bson:"providerName" json:"providerName"`
	CreatedAt         time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt         time.Time          `bson:"updatedAt" json:"updatedAt"`
}

