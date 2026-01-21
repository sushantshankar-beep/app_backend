package domain

import (
	"time"
)

type ProviderID string

type Provider struct {
	ID                   ProviderID   `bson:"_id,omitempty" json:"id"`
	ProviderCode  string     `bson:"providerCode" json:"providerCode"`
	Name             string `bson:"name" json:"name"`
	CompanyName      string `bson:"companyName" json:"companyName"`
	Email            string `bson:"email" json:"email"`
	Phone            string `bson:"phone" json:"phone"`
	AlternateContact string `bson:"alternateContact" json:"alternateContact"`
	ProfileURL       string `bson:"profileUrl" json:"profileUrl"`
	Address          string `bson:"address" json:"address"`
	PermanentAddress string `bson:"permanentAddress" json:"permanentAddress"`
	City             string `bson:"city" json:"city"`
	FcmToken         string `bson:"fcmToken" json:"fcmToken"`
	GSTNumber          string       `bson:"GSTNumber,omitempty" json:"gstNumber,omitempty"`
	VehicleNumber      string       `bson:"vehicleNumber,omitempty" json:"vehicleNumber,omitempty"`
	Description        string       `bson:"description,omitempty" json:"description,omitempty"`
	VehicleType        []string     `bson:"vehicleType,omitempty" json:"vehicleType,omitempty"`
	ProviderBrands     []string     `bson:"providerBrands,omitempty" json:"providerBrands,omitempty"`
	ProviderServices   []string     `bson:"providerServices,omitempty" json:"providerServices,omitempty"`
	IdentityProof      []Proof      `bson:"identityProof,omitempty" json:"identityProof,omitempty"`
	AddressProof       []Proof      `bson:"addressProof,omitempty" json:"addressProof,omitempty"`
	CancelCheque       CancelCheque `bson:"cancelCheque,omitempty" json:"cancelCheque,omitempty"`
	BankDetails        BankDetails  `bson:"bankDetails,omitempty" json:"bankDetails,omitempty"`

	FormSubmitted int       `bson:"formSubmitted" json:"formSubmitted"`
	IsActive      string    `bson:"isActive,omitempty" json:"isActive,omitempty"`
	Rating        string    `bson:"rating" json:"rating"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type Proof struct {
	Type     string `bson:"type,omitempty" json:"type,omitempty"`
	File     string `bson:"file,omitempty" json:"file,omitempty"`
	Verified string `bson:"verified,omitempty" json:"verified,omitempty"`

}

type CancelCheque struct {
	File     string `bson:"file,omitempty" json:"file,omitempty"`
	Verified string `bson:"verified,omitempty" json:"verified,omitempty"`
}

type BankDetails struct {
	AccountHolderName string `bson:"accountHolderName,omitempty" json:"accountHolderName,omitempty"`
	AccountNumber     string `bson:"accountNumber,omitempty" json:"accountNumber,omitempty"`
	IFSCCode          string `bson:"ifscCode,omitempty" json:"ifscCode,omitempty"`
	BranchName        string `bson:"branchName,omitempty" json:"branchName,omitempty"`
	UPI               string `bson:"upi,omitempty" json:"upi,omitempty"`
}


