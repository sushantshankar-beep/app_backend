package handlers

import (
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	svc *service.LocationService
}

func NewLocationHandler(svc *service.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

// =======================================
// USER LOCATION
// =======================================

func (h *LocationHandler) SaveUserLocation(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	var req domain.Location
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.UserID = userID

	if err := h.svc.SaveLocation(c, &req); err != nil {
		c.JSON(500, gin.H{"error": "failed to save location"})
		return
	}

	c.JSON(200, gin.H{"message": "location saved"})
}

func (h *LocationHandler) GetUserLocation(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)

	loc, err := h.svc.GetLocation(c, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch location"})
		return
	}

	c.JSON(200, loc)
}

// =======================================
// PROVIDER LOCATION
// =======================================

func (h *LocationHandler) SaveProviderLocation(c *gin.Context) {
	providerID := c.GetString(middleware.ContextKeyUserID)

	var req domain.Location
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	req.UserID = providerID

	if err := h.svc.SaveLocation(c, &req); err != nil {
		c.JSON(500, gin.H{"error": "failed to save location"})
		return
	}

	c.JSON(200, gin.H{"message": "location saved"})
}

func (h *LocationHandler) GetProviderLocation(c *gin.Context) {
	providerID := c.GetString(middleware.ContextKeyUserID)

	loc, err := h.svc.GetLocation(c, providerID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch location"})
		return
	}

	c.JSON(200, loc)
}
