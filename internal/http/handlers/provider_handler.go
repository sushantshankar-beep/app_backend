package handlers

import (
	"net/http"
    "log"
	"app_backend/internal/domain"
	"app_backend/internal/service"
	"strconv"
	"math"
	"app_backend/internal/http/middleware"
	"fmt"

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
	c.JSON(http.StatusOK, gin.H{
		"message": "OTP sent successfully",
	})
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

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"isNew": isNew,
	})
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile fetched successfully",
		"data":    p,
	})
}
func (h *ProviderHandler) CreateOrUpdateProfile(c *gin.Context) {
	id := c.GetString(middleware.ContextKeyUserID)
	fmt.Println("Provider ID from context for update:", id)

	pid := domain.ProviderID(id)

	log.Println("User ID from context:", id)
	log.Println("Provider ID from context:", pid)
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    updatedProfile,
	})
}

func (h *ProviderHandler) GetMyAllServices(c *gin.Context) {
    providerID := domain.ProviderID(c.GetString("userID"))

    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

    grouped, total, err := h.svc.GetMyAllServices(c, providerID, page, limit)
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "success": true,
        "data":    grouped,
        "pagination": gin.H{
            "currentPage": page,
            "totalPages":  int(math.Ceil(float64(total) / float64(limit))),
            "total":       total,
            "perPage":     limit,
        },
    })
}

func (h *ProviderHandler) GetMyService(c *gin.Context) {
    id := c.Param("id")
    providerID := domain.ProviderID(c.GetString("userID"))

    service, err := h.svc.GetMyService(c, providerID, id)
    if err != nil {
        c.JSON(404, gin.H{"success": false, "message": "Service not found"})
        return
    }

    c.JSON(200, gin.H{"success": true, "data": service})
}