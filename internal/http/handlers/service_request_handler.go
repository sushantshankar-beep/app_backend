package handlers

import (
	"log"
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"
	"app_backend/internal/validation"

	"github.com/gin-gonic/gin"
)

type ServiceRequestHandler struct {
	svc *service.ServiceRequestService
}

func NewServiceRequestHandler(s *service.ServiceRequestService) *ServiceRequestHandler {
	return &ServiceRequestHandler{svc: s}
}

func (h *ServiceRequestHandler) CreateServiceRequest(c *gin.Context) {
	var req validation.ServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)
	log.Println("Creating service request for user ID:", userID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	// userLocation, exists := c.Get("userLocation")
	// if !exists {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"success": false,
	// 		"message": "User location not found",
	// 	})
	// 	return
	// }

	// location, ok := userLocation.([]float64)
	// if !ok || len(location) != 2 {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"success": false,
	// 		"message": "Invalid user location format",
	// 	})
	// 	return
	// }

	serviceRequest, err := h.svc.CreateServiceRequest(c, userID, req)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Service request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create service request",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Service request created successfully",
		"data":    serviceRequest,
	})
}