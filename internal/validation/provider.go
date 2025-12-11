package validation

import (
	"errors"
	"regexp"
	"strings"
)

type ProviderProfileRequest struct {
	Name                string               `json:"name"`
	Email               string               `json:"email"`
	AlternateContact    string               `json:"alternateContact"`
	ProfileURL          string               `json:"profileUrl"`
	Address             string               `json:"address"`
	PermanentAddress    string               `json:"permanentAddress"`
	City                string               `json:"city"`
	GSTNumber           string               `json:"gstNumber"`
	VehicleNumber       string               `json:"vehicleNumber"`
	Description         string               `json:"description"`
	CompanyName         string               `json:"companyName"`
	PreferredLanguage   string               `json:"preferredLanguage"`
	TermsAndConditions  bool                 `json:"termsAndConditions"`
	VehicleType         []string             `json:"vehicleType"`
	ProviderServices    []string             `json:"providerServices"`
	ProviderBrands      []string             `json:"providerBrands"`
	IdentityProof       []ProofRequest       `json:"identityProof"`
	AddressProof        []ProofRequest       `json:"addressProof"`
	CancelCheque        *CancelChequeRequest `json:"cancelCheque"`
	BankDetails         *BankDetailsRequest  `json:"bankDetails"`
}

type ProofRequest struct {
	Type string `json:"type"`
	File string `json:"file"`
}

type CancelChequeRequest struct {
	File string `json:"file"`
}

type BankDetailsRequest struct {
	AccountHolderName string `json:"accountHolderName"`
	AccountNumber     string `json:"accountNumber"`
	IFSCCode          string `json:"ifscCode"`
	BranchName        string `json:"branchName"`
	UPI               string `json:"upi"`
}

func (r *ProviderProfileRequest) Validate() error {
	if r.Email != "" && !isValidEmail(r.Email) {
		return errors.New("invalid email format")
	}

	if r.AlternateContact != "" && !isValidPhone(r.AlternateContact) {
		return errors.New("invalid alternate contact format")
	}

	if r.GSTNumber != "" && !isValidGSTNumber(r.GSTNumber) {
		return errors.New("invalid GST number format")
	}

	if r.VehicleNumber != "" && strings.TrimSpace(r.VehicleNumber) == "" {
		return errors.New("vehicle number cannot be empty")
	}

	for i, proof := range r.IdentityProof {
		if strings.TrimSpace(proof.Type) == "" {
			return errors.New("identity proof type is required at index " + string(rune(i)))
		}
		if strings.TrimSpace(proof.File) == "" {
			return errors.New("identity proof file is required at index " + string(rune(i)))
		}
	}

	for i, proof := range r.AddressProof {
		if strings.TrimSpace(proof.Type) == "" {
			return errors.New("address proof type is required at index " + string(rune(i)))
		}
		if strings.TrimSpace(proof.File) == "" {
			return errors.New("address proof file is required at index " + string(rune(i)))
		}
	}

	if r.BankDetails != nil {
		if r.BankDetails.AccountNumber != "" && len(r.BankDetails.AccountNumber) < 9 {
			return errors.New("invalid account number")
		}
		if r.BankDetails.IFSCCode != "" && !isValidIFSC(r.BankDetails.IFSCCode) {
			return errors.New("invalid IFSC code format")
		}
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidPhone(phone string) bool {
	phoneRegex := regexp.MustCompile(`^[6-9]\d{9}$`)
	return phoneRegex.MatchString(phone)
}

func isValidGSTNumber(gst string) bool {
	gstRegex := regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
	return gstRegex.MatchString(gst)
}

func isValidIFSC(ifsc string) bool {
	ifscRegex := regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
	return ifscRegex.MatchString(ifsc)
}