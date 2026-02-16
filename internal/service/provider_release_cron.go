package service

import (
	"context"
	"log"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
)

/*
===================================================================
 PROVIDER ASSIGNMENT TIMEOUT CRON

 Runs every 1 min.

 If service is stuck in:
     status = provider_assigned
 for > 6 minutes:

  ✅ Cancel service
  ✅ Release provider
  ✅ Clear provider redis keys
  ✅ Unlock provider busy
  ✅ Restore geo
  ✅ Stop bidding loops
  ✅ Close provider socket room
===================================================================
*/

type ProviderAssignmentTimeoutCron struct {
	rdb          *redis.Client
	socket       *socket.Emitter
	acceptedRepo *repository.AcceptedServiceRepo
}

func NewProviderAssignmentTimeoutCron(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
) *ProviderAssignmentTimeoutCron {

	return &ProviderAssignmentTimeoutCron{
		rdb:          rdb,
		socket:       socket,
		acceptedRepo: acceptedRepo,
	}
}



//
// =======================
// START LOOP
// =======================
//

func StartProviderAssignmentTimeoutCron(
	ctx context.Context,
	cron *ProviderAssignmentTimeoutCron,
) {

	ticker := time.NewTicker(1 * time.Minute)

	go func() {

		for {
			select {

			case <-ticker.C:
				cron.Run(ctx)

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

//
// =======================
// MAIN SCAN
// =======================
//

func (c *ProviderAssignmentTimeoutCron) Run(ctx context.Context) {

	cutoff := time.Now().Add(-11 * time.Minute)

	cur, err := c.acceptedRepo.Col().Find(
		ctx,
		bson.M{
			"status": domain.StatusProviderAssigned,
			"updatedAt": bson.M{
				"$lt": cutoff,
			},
		},
	)

	if err != nil {
		log.Println("❌ provider-timeout cron mongo error:", err)
		return
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {

		var svc domain.AcceptedService
		if err := cur.Decode(&svc); err != nil {
			continue
		}

		go c.cancelService(ctx, &svc)
	}
}

//
// =======================
// CANCEL + CLEAN
// =======================
//

func (c *ProviderAssignmentTimeoutCron) cancelService(
	ctx context.Context,
	svc *domain.AcceptedService,
) {

	serviceID := svc.ID.Hex()
	providerID := svc.Provider.Hex()

	if providerID == "" {
		return
	}

	lockKey := "cron:providerAssignTimeout:" + serviceID

	ok, _ := c.rdb.SetNX(ctx, lockKey, "1", 3*time.Minute).Result()
	if !ok {
		return
	}
	defer c.rdb.Del(ctx, lockKey)

	log.Println("⏱ AUTO-CANCEL service:", serviceID, "provider:", providerID)

	// =====================================================
	// 🔒 STOP SEARCH / FLOWS
	// =====================================================

	c.rdb.Set(ctx, "service:stop:"+serviceID, "1", 30*time.Minute)
	c.rdb.Set(ctx, "service:locked:"+serviceID, "1", 30*time.Minute)

	// =====================================================
	// 🔓 RELEASE PROVIDER
	// =====================================================

	c.rdb.Del(ctx,
		"provider:busy:"+providerID,
	)

	// =====================================================
	// 🌍 RESTORE GEO
	// =====================================================

	if svc.ProviderLocation != nil {

		c.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
			Name:      providerID,
			Longitude: svc.ProviderLocation.Long,
			Latitude:  svc.ProviderLocation.Lat,
		})
	}

	// =====================================================
	// 🧹 REDIS CLEANUP
	// =====================================================

	c.cleanupKeys(ctx, serviceID, providerID)

	// =====================================================
	// ❌ CANCEL SERVICE IN DB
	// =====================================================

	now := time.Now()

	_, err := c.acceptedRepo.Col().UpdateByID(
		ctx,
		svc.ID,
		bson.M{
			"$set": bson.M{
				"status":      domain.StatusCancelled,
				"cancelledBy": "system_timeout",
				"cancelledAt": now,
				"updatedAt":   now,
				"reason":      "provider_not_confirmed",
			},
		},
	)

	if err != nil {
		log.Println("❌ cron cancel mongo failed:", err)
	}

	// =====================================================
	// 🚪 SOCKET CLEANUP
	// =====================================================

	c.socket.Emit(
		"provider:"+providerID,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
			"reason":    "auto_timeout",
		},
	)

	c.socket.CloseRoom("provider:" + providerID)

	c.socket.Emit(
		"user:"+serviceID,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
			"reason":    "provider_timeout",
		},
	)

	c.socket.CloseRoom("user:" + serviceID)

	log.Println("✅ AUTO-CANCEL DONE:", serviceID)
}

//
// =======================
// REDIS CLEANER
// =======================
//

func (c *ProviderAssignmentTimeoutCron) cleanupKeys(
	ctx context.Context,
	serviceID string,
	providerID string,
) {

	keys := []string{

		// provider
		"provider:busy:" + providerID,

		// provider scoped
		"service:providerBidLock:" + serviceID + ":" + providerID,
		"service:activeProvider:" + serviceID + ":" + providerID,
		"service:cooldown:" + serviceID + ":" + providerID,
		"service:sendcount:" + serviceID + ":" + providerID,
		"service:dist:" + serviceID + ":" + providerID,

		// service
		"service:bidWindow:" + serviceID,
	}

	c.rdb.Del(ctx, keys...)

	patterns := []string{
		"service:providerBidLock:" + serviceID + ":*",
		"service:activeProvider:" + serviceID + ":*",
		"service:cooldown:" + serviceID + ":*",
		"service:sendcount:" + serviceID + ":*",
		"service:dist:" + serviceID + ":*",
	}

	for _, p := range patterns {

		iter := c.rdb.Scan(ctx, 0, p, 500).Iterator()

		for iter.Next(ctx) {
			c.rdb.Del(ctx, iter.Val())
		}
	}
}
