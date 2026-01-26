package handlers

import (
	"net/http"

	"app_backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	repo *repository.DeviceTokenRepo
}

func NewDeviceHandler(repo *repository.DeviceTokenRepo) *DeviceHandler {
	return &DeviceHandler{
		repo: repo,
	}
}

// POST /devices/register
func (h *DeviceHandler) Register(c *gin.Context) {

	var req struct {
		Token string `json:"token" binding:"required"`
		Type  string `json:"type" binding:"required"` // user | provider
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token & type required"})
		return
	}

	var ownerID string

	if req.Type == "provider" {
		ownerID = c.GetString("providerId")
	} else {
		ownerID = c.GetString("userId")
	}

	if ownerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.repo.Save(
		c.Request.Context(),
		ownerID,
		req.Type,
		req.Token,
	); err != nil {

		c.JSON(500, gin.H{"error": "failed to save token"})
		return
	}

	c.JSON(200, gin.H{"status": "registered"})
}
