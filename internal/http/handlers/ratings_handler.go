package handlers

import (
	"net/http"

	"app_backend/internal/dto"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RatingHandler struct {
	svc *service.RatingService
}

func NewRatingHandler(svc *service.RatingService) *RatingHandler {
	return &RatingHandler{svc: svc}
}

func (h *RatingHandler) CreateProviderRating(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID is required"})
		return
	}

	var req dto.CreateRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BookingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "bookingId is required",
		})
		return
	}

	if req.Stars < 1 || req.Stars > 5 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "stars must be between 1 and 5",
		})
		return
	}

	if req.Recommended == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "recommended is required",
		})
		return
	}
	
	
	rating, err := h.svc.CreateProviderRating(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"rating": rating})
}

func (h *RatingHandler) CreateUserRating(c *gin.Context) {
	providerID := c.Param("providerID")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	var req dto.CreateRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BookingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "bookingId is required",
		})
		return
	}

	if req.Stars < 1 || req.Stars > 5 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "stars must be between 1 and 5",
		})
		return
	}

	if req.Recommended == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   true,
			"message": "recommended is required",
		})
		return
	}
	

	rating, err := h.svc.CreateUserRating(c.Request.Context(), providerID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"rating": rating})
}

func (h *RatingHandler) GetUserRatings(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userID is required"})
		return
	}

	ratings, err := h.svc.GetUserRatings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

func (h *RatingHandler) GetProviderRatings(c *gin.Context) {
	providerID := c.Param("providerID")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "providerID is required"})
		return
	}

	ratings, err := h.svc.GetProviderRatings(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}


