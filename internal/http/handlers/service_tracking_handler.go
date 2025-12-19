package handlers

import (
	// "net/http"

	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
)

type ServiceTrackingHandler struct {
	svc *service.ServiceTrackingService
}

func NewServiceTrackingHandler(
	svc *service.ServiceTrackingService,
) *ServiceTrackingHandler {
	return &ServiceTrackingHandler{svc: svc}
}

func (h *ServiceTrackingHandler) UserTracking(c *gin.Context) {
	serviceID := c.Param("id")

	data, err := h.svc.UserTrackingScreen(c.Request.Context(), serviceID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *ServiceTrackingHandler) ProviderTracking(c *gin.Context) {
	serviceID := c.Param("id")

	data, err := h.svc.ProviderTrackingScreen(c.Request.Context(), serviceID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

func (h *ServiceTrackingHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		OTP string `json:"otp"`
	}
	c.BindJSON(&req)

	serviceID := c.Param("id")

	if err := h.svc.VerifyOTP(c.Request.Context(), serviceID, req.OTP); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "verified"})
}
