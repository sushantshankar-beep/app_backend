package handlers

import (
	"net/http"
    "strings"
	"app_backend/internal/domain"
	"app_backend/internal/s3"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"app_backend/internal/validation"
)

type KYCHandler struct {
	svc *service.KYCService
}

func NewKYCHandler(svc *service.KYCService) *KYCHandler {
	return &KYCHandler{svc: svc}
}

func (h *KYCHandler) CreateOrUpdateKYC(c *gin.Context) {

	providerID := c.GetString("providerId")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Force multipart parsing
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	ct := c.GetHeader("Content-Type")
	if !strings.Contains(ct, "multipart/form-data") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be multipart/form-data"})
		return
	}

	// Upload files
	aadhaarFront, _ := s3.GetUploadedURLs(c, "aadhaarFront")
	aadhaarBack, _ := s3.GetUploadedURLs(c, "aadhaarBack")
	pan, _ := s3.GetUploadedURLs(c, "pan")

	accountHolderName := strings.TrimSpace(c.PostForm("accountHolderName"))
	accountNumber := strings.TrimSpace(c.PostForm("accountNumber"))
	ifsc := strings.ToUpper(strings.TrimSpace(c.PostForm("ifsc")))
	branchName := strings.TrimSpace(c.PostForm("branchName"))
	upiID := strings.TrimSpace(c.PostForm("upiId"))
	gstNumber := strings.ToUpper(strings.TrimSpace(c.PostForm("gstNumber")))

	requiredStrings := map[string]string{
		"accountHolderName": accountHolderName,
		"accountNumber":     accountNumber,
		"ifsc":              ifsc,
		"branchName":        branchName,
	}
	for field, val := range requiredStrings {
		if val == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": field + " is required"})
			return
		}
	}
	
	requiredFiles := map[string][]string{
		"aadhaarFront": aadhaarFront,
		"aadhaarBack":  aadhaarBack,
		"pan":          pan,
	}
	for field, urls := range requiredFiles {
		if len(urls) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": field + " is required"})
			return
		}
	}

	if !validation.IsValidAccountNumber(accountNumber) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account number"})
		return
	}

	if !validation.IsValidIFSC(ifsc) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid IFSC"})
		return
	}

	if gstNumber != "" && !validation.IsValidGST(gstNumber) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid GST number"})
		return
	}

	if upiID != "" && !validation.IsValidUPI(upiID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid UPI ID"})
		return
	}

	// Build KYC
	kyc := &domain.ProviderKYC{
		Bank: domain.ProviderBankDetails{
			AccountHolderName: accountHolderName,
			AccountNumber:     accountNumber,
			IFSC:              ifsc,
			BranchName:        branchName,
			UPIID:             upiID,
			GSTNumber:         gstNumber,
		},
	}

	if len(aadhaarFront) > 0 {
		kyc.Documents = append(kyc.Documents,
			domain.KYCDocument{
				Type: domain.DOC_AADHAAR_FRONT,
				URL:  aadhaarFront[0],
			})
	}

	if len(aadhaarBack) > 0 {
		kyc.Documents = append(kyc.Documents,
			domain.KYCDocument{
				Type: domain.DOC_AADHAAR_BACK,
				URL:  aadhaarBack[0],
			})
	}

	if len(pan) > 0 {
		kyc.Documents = append(kyc.Documents,
			domain.KYCDocument{
				Type: domain.DOC_PAN,
				URL:  pan[0],
			})
	}

	result, err := h.svc.CreateOrUpdateKYC(
		c.Request.Context(),
		providerID,
		kyc,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "KYC submitted successfully",
		"data":    result,
	})
}


func (h *KYCHandler) GetKYC(c *gin.Context) {
	providerID := c.GetString("providerId")

	kyc, err := h.svc.GetKYC(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "KYC retrieved successfully",
		"data":    kyc,
	})
}