package handlers

import (
	"net/http"

	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
	"os"
)

type ZoneHandler struct {
	zoneService *service.ZoneService
}

func NewZoneHandler(s *service.ZoneService) *ZoneHandler {
	return &ZoneHandler{zoneService: s}
}

type CheckZoneRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}
// config.go
var DisableZoneCheck = true

func (h *ZoneHandler) CheckServiceable(c *gin.Context) {

	var req CheckZoneRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	// 🔥 Temporary bypass using ENV variable
	if os.Getenv("DISABLE_ZONE_CHECK") == "true" {
		c.JSON(http.StatusOK, gin.H{
			"serviceable": true,
			"zoneName":    "All India (Temporary)",
		})
		return
	}

	// Normal flow
	zone, err := h.zoneService.CheckServiceable(
		c.Request.Context(),
		req.Latitude,
		req.Longitude,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Server error",
		})
		return
	}

	if zone == nil {
		c.JSON(http.StatusOK, gin.H{
			"serviceable": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"serviceable": true,
		"zoneName":    zone.ZoneName,
	})
}