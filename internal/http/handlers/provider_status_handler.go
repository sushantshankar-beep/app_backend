package handlers

import (
	"net/http"
	// "time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"app_backend/internal/repository"
	"app_backend/internal/domain"
	"app_backend/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "context"
	"log"
	"go.mongodb.org/mongo-driver/bson"
	"time"
	"context"
)

type ProviderStatusHandler struct {
	redis *redis.Client
	providerRepo *repository.ProviderRepo
	acceptedRepo  *repository.AcceptedServiceRepo
	biddingSvc    *service.BiddingService
}

func NewProviderStatusHandler(
	r *redis.Client,
	providerRepo *repository.ProviderRepo,
	acceptedRepo  *repository.AcceptedServiceRepo,
	biddingSvc    *service.BiddingService,
) *ProviderStatusHandler {

	return &ProviderStatusHandler{
		redis:        r,
		providerRepo: providerRepo,
		acceptedRepo: acceptedRepo,
		biddingSvc: biddingSvc,
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Load provider
	provider, err := h.providerRepo.FindByID(ctx, domain.ProviderID(providerID))
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
	// 🔥 Redis Pipeline (1 round trip)
	// ===========================
	pipe := h.redis.Pipeline()

	pipe.Set(ctx, "provider:online:"+providerID, "1", 0)

	pipe.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
		Name:      providerID,
		Longitude: req.Lng,
		Latitude:  req.Lat,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "redis error"})
		return
	}

	// Update Mongo
	if err := h.providerRepo.SetOnlineStatus(
		ctx,
		domain.ProviderID(providerID),
		true,
		req.Lat,
		req.Lng,
	); err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "db update failed"})
		return
	}

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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 6*time.Second)
	defer cancel()

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
		return
	}

	// ============================================
	// 🔍 Check Active Service
	// ============================================

	activeSvc, err := h.acceptedRepo.FindActiveServiceByProvider(ctx, providerOID)

	if err == nil && activeSvc != nil {

		if activeSvc.Status == domain.StatusProviderAssigned {

			_, err := h.acceptedRepo.Col().UpdateByID(
				ctx,
				activeSvc.ID,
				bson.M{
					"$set": bson.M{
						"provider":            primitive.NilObjectID,
						"cancelledProviderID": providerID,
						"updatedAt":           time.Now(),
					},
				},
			)

			if err != nil {
				log.Println("remove provider failed:", err)
			}
		}

		// Clean service-level Redis keys
		pipe := h.redis.Pipeline()
		pipe.Del(ctx, "service:locked:"+activeSvc.ID.Hex())
		pipe.Del(ctx, "service:stop:"+activeSvc.ID.Hex())
		pipe.Exec(ctx)
	}

	// ============================================
	// 🔥 Provider Redis Cleanup (Pipeline)
	// ============================================

	pipe := h.redis.Pipeline()
	pipe.Del(ctx, "provider:online:"+providerID)
	pipe.ZRem(ctx, "providers:geo", providerID)
	pipe.Del(ctx, "provider:busy:"+providerID)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Println("redis cleanup error:", err)
	}

	// ============================================
	// 🗄 Update Mongo
	// ============================================

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

	c.JSON(http.StatusOK, gin.H{
		"status": "offline",
	})
}
