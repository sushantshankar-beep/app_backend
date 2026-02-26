package handlers

import (
	"net/http"

	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"app_backend/internal/domain"
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

	serviceID := c.Param("id")

	var req struct {
		OTP  string  `json:"otp"`
		Lat  float64 `json:"lat"`
		Long float64 `json:"long"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.svc.VerifyOTP(
		c.Request.Context(),
		serviceID,
		req.OTP,
		req.Lat,
		req.Long,
	)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	resp["status"] = "verified"

	c.JSON(200, resp)
}

func (h *ServiceTrackingHandler) UpdateStatus(c *gin.Context) {

	serviceID := c.Param("id")

	var req struct {
		Status string  `json:"status"`
		Lat    float64 `json:"lat"`
		Long   float64 `json:"long"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	resp, err := h.svc.UpdateStatus(
		c.Request.Context(),
		serviceID,
		domain.ServiceStatus(req.Status),
		req.Lat,
		req.Long,
	)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resp)
}

func (h *ServiceTrackingHandler) GetInvoiceUrl(c *gin.Context) {
	serviceID := c.Query("bookingId")

	if serviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "serviceId is required",
		})
		return
	}

	invoice, err := h.svc.GetInvoiceUrl(c.Request.Context(), serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invoice fetched successfully",
		"data": gin.H{
			"invoiceUrl": invoice.PDFUrl,
		},
	})
}
func (h *ServiceTrackingHandler) UpdateLiveLocation(c *gin.Context) {

	serviceID := c.Param("serviceId")

	var req struct {
		Lat  float64 `json:"lat"`
		Long float64 `json:"long"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	err := h.svc.UpdateLiveLocation(
		c.Request.Context(),
		serviceID,
		req.Lat,
		req.Long,
	)

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"success": true})
}
