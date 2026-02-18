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

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider"})
		return
	}

	// ============================================
	// 🔍 CHECK ACTIVE SERVICE
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
				log.Println("❌ remove provider failed:", err)
			}
		}

		// 🔓 Clean service redis keys SAFELY
		h.redis.Del(ctx, "service:locked:"+activeSvc.ID.Hex())
		h.redis.Del(ctx, "service:stop:"+activeSvc.ID.Hex())
	}

	// ============================================
	// 🧹 REDIS CLEANUP (Provider level)
	// ============================================

	h.redis.Del(ctx, "provider:online:"+providerID)
	h.redis.ZRem(ctx, "providers:geo", providerID)
	h.redis.Del(ctx, "provider:busy:"+providerID)

	// ============================================
	// 🗄 UPDATE MONGO
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
