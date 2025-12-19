package handlers

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
)

type BiddingHandler struct {
	svc *service.BiddingService
}

func NewBiddingHandler(svc *service.BiddingService) *BiddingHandler {
	return &BiddingHandler{svc}
}

func (h *BiddingHandler) FindMechanics(c *gin.Context) {
	var req struct {
		ServiceID string `json:"serviceId"`
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	c.ShouldBindJSON(&req)

	go h.svc.FindMechanics(
		c.Request.Context(),
		req.ServiceID,
		req.Lat,
		req.Lng,
	)

	c.JSON(http.StatusOK, gin.H{"status": "search_started"})
}

func (h *BiddingHandler) PlaceBid(c *gin.Context) {
	var req struct {
		ServiceID string `json:"serviceId"`
		Price float64 `json:"price"`
	}
	c.ShouldBindJSON(&req)

	providerID := c.GetString("providerId")

	if err := h.svc.PlaceBid(
		c.Request.Context(),
		req.ServiceID,
		providerID,
		req.Price,
	); err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "bid_sent"})
}

func (h *BiddingHandler) AcceptBid(c *gin.Context) {
	var req struct {
		ServiceID string `json:"serviceId"`
		ProviderID string `json:"providerId"`
	}
	c.ShouldBindJSON(&req)

	if err := h.svc.AcceptBid(
		c.Request.Context(),
		req.ServiceID,
		req.ProviderID,
	); err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "bid_accepted"})
}
