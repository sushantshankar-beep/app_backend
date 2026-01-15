package handlers

import (
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type KYCHandler struct {
	svc *service.KYCService
}

func NewKYCHandler(svc *service.KYCService) *KYCHandler {
	return &KYCHandler{svc: svc}
}

func (h *KYCHandler) SubmitKYC(c *gin.Context) {
	providerID := c.GetString("providerId")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.ProviderKYC
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.SubmitKYC(c.Request.Context(), providerID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "KYC_SUBMITTED"})
}

func (h *KYCHandler) GetKYC(c *gin.Context) {
	providerID := c.GetString("providerId")

	kyc, err := h.svc.GetKYC(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "KYC not found"})
		return
	}

	c.JSON(http.StatusOK, kyc)
}
