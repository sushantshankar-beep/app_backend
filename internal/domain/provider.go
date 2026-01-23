package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type ProviderID string

type Provider struct {
	ID                   ProviderID         `bson:"_id,omitempty" json:"id"`
	ProviderCode         string             `bson:"providerCode" json:"providerCode"`
	Name                 string             `bson:"name" json:"name"`
	CompanyName          string             `bson:"companyName" json:"companyName"`
	Email                string             `bson:"email" json:"email"`
	Phone                string             `bson:"phone" json:"phone"`
	AlternateContact     string             `bson:"alternateContact" json:"alternateContact"`
	ProfileURL           string             `bson:"profileUrl" json:"profileUrl"`
	Address              string             `bson:"address" json:"address"`
	PermanentAddress     string             `bson:"permanentAddress" json:"permanentAddress"`
	City                 string             `bson:"city" json:"city"`
	FcmToken             string             `bson:"fcmToken" json:"fcmToken"`
	VehicleNumber        string             `bson:"vehicleNumber,omitempty" json:"vehicleNumber,omitempty"`
	Description          string             `bson:"description,omitempty" json:"description,omitempty"`
	VehicleType          []string           `bson:"vehicleType,omitempty" json:"vehicleType,omitempty"`
	ProviderBrands       []string           `bson:"providerBrands,omitempty" json:"providerBrands,omitempty"`
	ProviderServices     []string           `bson:"providerServices,omitempty" json:"providerServices,omitempty"`
	KYCID                primitive.ObjectID `bson:"kycId,omitempty" json:"kycId,omitempty"`
	FormSubmitted        int                `bson:"formSubmitted" json:"formSubmitted"`
	IsAgreementSubmitted bool               `bson:"isAgreementSubmitted" json:"isAgreementSubmitted"`
	AgreementSubmittedAt *time.Time          `bson:"agreementSubmittedAt,omitempty" json:"agreementSubmittedAt,omitempty"`
	AgreementPDF         string             `bson:"agreementPdf,omitempty" json:"agreementPdf,omitempty"`
	CommissionPercentage float64            `bson:"commissionPercentage,omitempty" json:"commissionPercentage,omitempty"`
	IsActive             string             `bson:"isActive,omitempty" json:"isActive,omitempty"`
	Rating               string             `bson:"rating" json:"rating"`
	CreatedAt            time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time          `bson:"updatedAt" json:"updatedAt"`
}

