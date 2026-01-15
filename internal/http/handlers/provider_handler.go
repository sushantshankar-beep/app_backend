package handlers

import (
	"math"
	"net/http"
	"strconv"

	"app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderHandler struct {
	svc *service.ProviderService
}

func NewProviderHandler(s *service.ProviderService) *ProviderHandler {
	return &ProviderHandler{svc: s}
}

/* ---------------- OTP ---------------- */

func (h *ProviderHandler) SendOTP(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}

	if err := h.svc.SendOTP(c.Request.Context(), req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
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

	token, isNew, err := h.svc.VerifyOTP(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"isNew": isNew,
	})
}

/* ---------------- PROFILE ---------------- */

func (h *ProviderHandler) Profile(c *gin.Context) {
	providerObjID := c.MustGet(middleware.ContextKeyProviderObjID).(primitive.ObjectID)
	providerID := domain.ProviderID(providerObjID.Hex())

	p, err := h.svc.GetProfile(c.Request.Context(), providerID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
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
	providerObjID := c.MustGet(middleware.ContextKeyProviderObjID).(primitive.ObjectID)
	providerID := domain.ProviderID(providerObjID.Hex())

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := h.svc.CreateOrUpdateProfile(
		c.Request.Context(),
		providerID,
		req,
	)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    updated,
	})
}

/* ---------------- SERVICES ---------------- */

func (h *ProviderHandler) GetMyAllServices(c *gin.Context) {
	providerObjID := c.MustGet(middleware.ContextKeyProviderObjID).(primitive.ObjectID)
	providerID := domain.ProviderID(providerObjID.Hex())

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	grouped, total, err := h.svc.GetMyAllServices(
		c.Request.Context(),
		providerID,
		page,
		limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    grouped,
		"pagination": gin.H{
			"currentPage": page,
			"totalPages":  totalPages,
			"total":       total,
			"perPage":     limit,
		},
	})
}

func (h *ProviderHandler) GetMyService(c *gin.Context) {
	serviceID := c.Param("id")

	providerObjID := c.MustGet(middleware.ContextKeyProviderObjID).(primitive.ObjectID)
	providerID := domain.ProviderID(providerObjID.Hex())

	svc, err := h.svc.GetMyService(
		c.Request.Context(),
		providerID,
		serviceID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Service not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    svc,
	})
}
