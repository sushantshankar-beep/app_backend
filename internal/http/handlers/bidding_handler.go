package handlers

import (
	"app_backend/internal/domain"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type BiddingHandler struct {
	svc *service.BiddingService
}

func NewBiddingHandler(svc *service.BiddingService) *BiddingHandler {
	return &BiddingHandler{svc: svc}
}

/*
POST /bidding/find
*/
func (h *BiddingHandler) FindMechanics(c *gin.Context) {
	var req struct {
		Lat         float64  `json:"lat" binding:"required"`
		Lng         float64  `json:"lng" binding:"required"`
		ServiceType string   `json:"serviceType" binding:"required"`
		Issues      []string `json:"issues"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	serviceID, err := h.svc.StartSearch(
		c.Request.Context(),
		domain.UserID(userID),
		req.ServiceType,
		req.Issues,
		req.Lat,
		req.Lng,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "search_started",
		"serviceId": serviceID,
	})
}

/*
POST /bidding/bid
*/
func (h *BiddingHandler) PlaceBid(c *gin.Context) {
	var req struct {
		ServiceID string  `json:"serviceId" binding:"required"`
		Price     float64 `json:"price" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	providerID := c.GetString("providerId")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	bidID,err := h.svc.PlaceBid(c.Request.Context(),req.ServiceID,providerID,req.Price)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "bid_sent",
		"bidId":  bidID,
	})
}

/*
POST /bidding/accept
*/
func (h *BiddingHandler) AcceptBid(c *gin.Context) {
	var req struct {
		ServiceID  string  `json:"serviceId" binding:"required"`
		BidID      string  `json:"bidId" binding:"required"`
		ProviderID string  `json:"providerId" binding:"required"`
		Price      float64 `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.svc.AcceptBid(c.Request.Context(),req.ServiceID,req.BidID,req.ProviderID,req.Price); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "bid_accepted",
	})
}
