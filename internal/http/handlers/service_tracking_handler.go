package handlers

import (
	// "net/http"

	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"app_backend/internal/domain"
	"fmt"
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
		Lat float64 `json:"lat"`
		Long float64 `json:"long"`
	}
	c.BindJSON(&req)

	serviceID := c.Param("id")
	fmt.Println("otp through handler",req.OTP)

	if err := h.svc.VerifyOTP(c.Request.Context(), serviceID, req.OTP,req.Lat,req.Long); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "verified"})
}
func (h *ServiceTrackingHandler) UpdateStatus(c *gin.Context) {
	serviceID := c.Param("id")

	var req struct {
		Status string `json:"status"`
		Lat    float64 `json:"lat"`
		Long   float64  `json:"long"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.svc.UpdateStatus(
		c.Request.Context(),
		serviceID,
		domain.ServiceStatus(req.Status),
		req.Lat,req.Long,
	); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"status": "updated",
	})
}

