package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AgreementHandler struct {
	svc *service.AgreementService
}

func NewAgreementHandler(svc *service.AgreementService) *AgreementHandler {
	return &AgreementHandler{svc: svc}
}

func (h *AgreementHandler) GetProviderAgreement(c *gin.Context) {

	providerObjID := c.MustGet(middleware.ContextKeyProviderObjID).(primitive.ObjectID)
	providerID := domain.ProviderID(providerObjID.Hex())
	
	agreement, err := h.svc.GetProviderAgreement(c, providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agreement)
}