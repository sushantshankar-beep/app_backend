package domain

import "time"

type ProviderID string

type Provider struct {
	ID ProviderID `bson:"_id,omitempty" json:"id"`
	Slug           string `bson:"slug,omitempty" json:"slug,omitempty"`
	AppStateStatus string `bson:"appStateStatus,omitempty" json:"appStateStatus,omitempty"`
	Name             string `bson:"name,omitempty" json:"name,omitempty"`
	Phone            string `bson:"phone" json:"phone"`
	Email            string `bson:"email,omitempty" json:"email,omitempty"`
	AlternateContact string `bson:"alternateContact,omitempty" json:"alternateContact,omitempty"`
	ProfileURL       string `bson:"profileUrl,omitempty" json:"profileUrl,omitempty"`
	Location GeoLocation `bson:"location,omitempty" json:"location,omitempty"`
	Address          string `bson:"address,omitempty" json:"address,omitempty"`
	PermanentAddress string `bson:"permanentAddress,omitempty" json:"permanentAddress,omitempty"`
	Status string `bson:"status,omitempty" json:"status,omitempty"`
	GSTNumber string `bson:"GSTNumber,omitempty" json:"gstNumber,omitempty"`
	IdentityProof []Proof `bson:"identityProof,omitempty" json:"identityProof,omitempty"`
	AddressProof  []Proof `bson:"addressProof,omitempty" json:"addressProof,omitempty"`
	// CancelCheque CancelCheque `bson:"cancelCheque,omitempty" json:"cancelCheque,omitempty"`
	BankDetails BankDetails `bson:"bankDetails,omitempty" json:"bankDetails,omitempty"`
	VehicleNumber string `bson:"vehicleNumber,omitempty" json:"vehicleNumber,omitempty"`
	FormSubmitted int    `bson:"formSubmitted,omitempty" json:"formSubmitted"`
	IsAssigned    bool   `bson:"isAssigned,omitempty" json:"isAssigned,omitempty"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	VehicleType      []string `bson:"vehicleType,omitempty" json:"vehicleType,omitempty"`
	ProviderBrands   []string `bson:"providerBrands,omitempty" json:"providerBrands,omitempty"`
	ProviderServices []string `bson:"providerServices,omitempty" json:"providerServices,omitempty"`
	CompanyName string `bson:"companyName,omitempty" json:"companyName,omitempty"`
	City        string `bson:"city,omitempty" json:"city,omitempty"`
	PreferredLanguage string `bson:"preferredLanguage,omitempty" json:"preferredLanguage,omitempty"`
	TermsAndConditions bool `bson:"termsAndConditions,omitempty" json:"termsAndConditions,omitempty"`
	FCMToken string `bson:"fcmToken,omitempty" json:"fcmToken,omitempty"`
	IsSocketConnected bool `bson:"isSocketConnected,omitempty" json:"isSocketConnected,omitempty"`
	IsServiceOn       bool `bson:"isServiceOn,omitempty" json:"isServiceOn,omitempty"`
	Tokens []string `bson:"tokens,omitempty" json:"tokens,omitempty"`
	IsActive string `bson:"isActive,omitempty" json:"isActive,omitempty"`
	CommissionPercentage float64 `bson:"commissionPercentage,omitempty" json:"commissionPercentage,omitempty"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type GeoLocation struct {
	Type        string    `bson:"type,omitempty" json:"type,omitempty"`
	Coordinates []float64 `bson:"coordinates,omitempty" json:"coordinates,omitempty"`
}

type Proof struct {
	Type     string `bson:"type,omitempty" json:"type,omitempty"`
	File     string `bson:"file,omitempty" json:"file,omitempty"`
	Verified bool `bson:"verified,omitempty" json:"verified,omitempty"`
}

// type CancelCheque struct {
// 	File     string `bson:"file,omitempty" json:"file,omitempty"`
// 	Verified bool `bson:"verified,omitempty" json:"verified,omitempty"`
// }

type BankDetails struct {
	AccountHolderName string `bson:"accountHolderName,omitempty" json:"accountHolderName,omitempty"`
	AccountNumber     string `bson:"accountNumber,omitempty" json:"accountNumber,omitempty"`
	IFSCCode          string `bson:"ifscCode,omitempty" json:"ifscCode,omitempty"`
	BranchName        string `bson:"branchName,omitempty" json:"branchName,omitempty"`
	UPI               string `bson:"upi,omitempty" json:"upi,omitempty"`
}
