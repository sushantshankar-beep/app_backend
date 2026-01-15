package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type ProviderStatusHandler struct {
	redis *redis.Client
}

func NewProviderStatusHandler(r *redis.Client) *ProviderStatusHandler {
	return &ProviderStatusHandler{redis: r}
}

/*
POST /provider/online
Authorization: Bearer PROVIDER_TOKEN
*/
func (h *ProviderStatusHandler) GoOnline(c *gin.Context) {
	providerID := c.GetString("providerId")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Lat float64 `json:"lat" binding:"required"`
		Lng float64 `json:"lng" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat & lng required"})
		return
	}

	ctx := c.Request.Context()
	if err := h.redis.Set(ctx,"provider:online:"+providerID,"1",5*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
		return
	}
	if err := h.redis.GeoAdd(ctx,"providers:geo",
		&redis.GeoLocation{
			Name:      providerID,
			Longitude: req.Lng,
			Latitude:  req.Lat,
		},
	).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "geo error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "online",
		"room":   "provider:" + providerID,
		"wsUrl":  "/ws?room=provider:" + providerID,
	})
}



/*
POST /provider/offline
*/
func (h *ProviderStatusHandler) GoOffline(c *gin.Context) {
	providerID := c.GetString("providerId")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	h.redis.Del(c.Request.Context(), "provider:online:"+providerID)

	c.JSON(http.StatusOK, gin.H{
		"status": "offline",
	})
}
