package domain

import (
	"time"
)

type ProviderID string

type Provider struct {
	ID                   ProviderID   `bson:"_id,omitempty" json:"id"`
	Services             []string     `bson:"services"`
	Name                 string       `bson:"name,omitempty" json:"name,omitempty"`
	Phone                string       `bson:"phone" json:"phone"`
	Email                string       `bson:"email,omitempty" json:"email,omitempty"`
	AlternateContact     string       `bson:"alternateContact,omitempty" json:"alternateContact,omitempty"`
	ProfileURL           string       `bson:"profileUrl,omitempty" json:"profileUrl,omitempty"`
	Address              string       `bson:"address,omitempty" json:"address,omitempty"`
	PermanentAddress     string       `bson:"permanentAddress,omitempty" json:"permanentAddress,omitempty"`
	City                 string       `bson:"city,omitempty" json:"city,omitempty"`
	Status               string       `bson:"status,omitempty" json:"status,omitempty"`
	GSTNumber            string       `bson:"GSTNumber,omitempty" json:"gstNumber,omitempty"`
	IdentityProof        []Proof      `bson:"identityProof,omitempty" json:"identityProof,omitempty"`
	AddressProof         []Proof      `bson:"addressProof,omitempty" json:"addressProof,omitempty"`
	CancelCheque         CancelCheque `bson:"cancelCheque,omitempty" json:"cancelCheque,omitempty"`
	BankDetails          BankDetails  `bson:"bankDetails,omitempty" json:"bankDetails,omitempty"`
	VehicleNumber        string       `bson:"vehicleNumber,omitempty" json:"vehicleNumber,omitempty"`
	FormSubmitted        int          `bson:"formSubmitted" json:"formSubmitted"`
	Description          string       `bson:"description,omitempty" json:"description,omitempty"`
	VehicleType          []string     `bson:"vehicleType,omitempty" json:"vehicleType,omitempty"`
	ProviderBrands       []string     `bson:"providerBrands,omitempty" json:"providerBrands,omitempty"`
	ProviderServices     []string     `bson:"providerServices,omitempty" json:"providerServices,omitempty"`
	CompanyName          string       `bson:"companyName,omitempty" json:"companyName,omitempty"`
	IsActive             string        `bson:"isActive,omitempty" json:"isActive,omitempty"`
	CommissionPercentage float64      `bson:"commissionPercentage" json:"commissionPercentage"`
	CreatedAt            time.Time    `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time    `bson:"updatedAt" json:"updatedAt"`
}

type Proof struct {
	Type     string `bson:"type,omitempty" json:"type,omitempty"`
	File     string `bson:"file,omitempty" json:"file,omitempty"`
	Verified  string `bson:"verified,omitempty" json:"verified,omitempty"`
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
