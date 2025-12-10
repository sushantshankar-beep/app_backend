package handlers

import (
	"net/http"

	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ProviderHandler struct {
	svc *service.ProviderService
}

func NewProviderHandler(s *service.ProviderService) *ProviderHandler {
	return &ProviderHandler{svc: s}
}

func (h *ProviderHandler) SendOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}
	if err := h.svc.SendOTP(c, req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProviderHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, isNew, err := h.svc.VerifyOTP(c, req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "isNew": isNew})
}

func (h *ProviderHandler) Profile(c *gin.Context) {
	id := c.GetString(middleware.ContextKeyUserID)
	pid := domain.ProviderID(id)

	p, err := h.svc.GetProfile(c, pid)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *ProviderHandler) CreateOrUpdateProfile(c *gin.Context) {
	id := c.GetString(middleware.ContextKeyUserID)
	pid := domain.ProviderID(id)

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	updatedProfile, err := h.svc.CreateOrUpdateProfile(c, pid, req)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedProfile)
}

// func (h *ProviderHandler) Dashboard(c *gin.Context) {
// 	id := c.GetString(middleware.ContextKeyUserID)
// 	providerObjID, err := primitive.ObjectIDFromHex(id)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
// 		return
// 	}

// 	stats, err := h.svc.GetDashboardStats(c, providerObjID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"status":  200,
// 		"message": "Dashboard stats fetched successfully",
// 		"data":    stats,
// 	})
// }