package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
)

type AgreementHandler struct {
	svc *service.AgreementService
}

func NewAgreementHandler(svc *service.AgreementService) *AgreementHandler {
	return &AgreementHandler{
		svc: svc,
	}
}

func (h *AgreementHandler) GetAgreement(c *gin.Context) {
	id := c.Param("id")

	safeHTML := c.Query("safe") == "true"

	if safeHTML {
		agreement, err := h.svc.GetAgreementSafeHTML(c, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusOK, agreement)
		return
	}

	agreement, err := h.svc.GetAgreement(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, agreement)
}