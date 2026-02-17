package handlers

import (
	"net/http"
	// "time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"app_backend/internal/repository"
	"app_backend/internal/domain"
)

type ProviderStatusHandler struct {
	redis *redis.Client
	providerRepo *repository.ProviderRepo
}

func NewProviderStatusHandler(
	r *redis.Client,
	providerRepo *repository.ProviderRepo,
) *ProviderStatusHandler {

	return &ProviderStatusHandler{
		redis:        r,
		providerRepo: providerRepo,
	}
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

	ctx := c.Request.Context()

	// ===========================
	// 🔍 LOAD PROVIDER FROM DB
	// ===========================

	provider, err := h.providerRepo.FindByID(
		ctx,
		domain.ProviderID(providerID),
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "provider not found"})
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

	// ===========================
	// 📍 REDIS ONLINE + GEO
	// ===========================

	if err := h.redis.Set(ctx,
		"provider:online:"+providerID,
		"1",
		0,
	).Err(); err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
		return
	}

	if err := h.redis.GeoAdd(ctx,
		"providers:geo",
		&redis.GeoLocation{
			Name:      providerID,
			Longitude: req.Lng,
			Latitude:  req.Lat,
		},
	).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "geo error"})
		return
	}
	if err := h.providerRepo.SetOnlineStatus(ctx,domain.ProviderID(providerID),true,req.Lat,req.Lng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}



	// ===========================
	// ✅ RESPONSE
	// ===========================

	c.JSON(http.StatusOK, gin.H{
		"status":               "online",
		"isAgreementSubmitted": provider.IsAgreementSubmitted,
		"room":                 "provider:" + providerID,
		"wsUrl":                "/ws?room=provider:" + providerID,
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
	ctx := c.Request.Context()
	// 🔍 Optional: block offline if active service
	busy, _ := h.redis.Exists(ctx, "provider:busy:"+providerID).Result()
	if busy == 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "You Already Have Active Service",
		})
		return
	}

	// ===========================
	// 🧹 REDIS CLEANUP
	// ===========================

	h.redis.Del(ctx, "provider:online:"+providerID)
	h.redis.ZRem(ctx, "providers:geo", providerID)
	h.redis.Del(ctx, "provider:busy:"+providerID)

	// ===========================
	// 🗄 UPDATE MONGO
	// ===========================

	if err := h.providerRepo.SetOnlineStatus(
		ctx,
		domain.ProviderID(providerID),
		false,
		0,
		0,
	); err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

	// ===========================
	// ✅ RESPONSE
	// ===========================

	c.JSON(http.StatusOK, gin.H{
		"status": "offline",
	})
}
