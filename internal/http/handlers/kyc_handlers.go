package handlers

import (
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/s3"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
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

	aadhaarFront, _ := s3.GetUploadedURLs(c, "aadhaarFront")
	aadhaarBack, _ := s3.GetUploadedURLs(c, "aadhaarBack")
	pan, _ := s3.GetUploadedURLs(c, "pan")

	accountHolderName := c.PostForm("accountHolderName")
	accountNumber := c.PostForm("accountNumber")
	ifsc := c.PostForm("ifsc")
	branchName := c.PostForm("branchName")
	upiID := c.PostForm("upiId")
	gstNumber := c.PostForm("gstNumber")

	kyc := &domain.ProviderKYC{
		Documents: []domain.KYCDocument{},
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
		kyc.Documents = append(kyc.Documents, domain.KYCDocument{
			Type: domain.DOC_AADHAAR_FRONT,
			URL:  aadhaarFront[0],
		})
	}

	if len(aadhaarBack) > 0 {
		kyc.Documents = append(kyc.Documents, domain.KYCDocument{
			Type: domain.DOC_AADHAAR_BACK,
			URL:  aadhaarBack[0],
		})
	}

	if len(pan) > 0 {
		kyc.Documents = append(kyc.Documents, domain.KYCDocument{
			Type: domain.DOC_PAN,
			URL:  pan[0],
		})
	}

	result, err := h.svc.CreateOrUpdateKYC(c.Request.Context(), providerID, kyc)
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