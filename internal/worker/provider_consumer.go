package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/events"
	"app_backend/internal/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderConsumer struct {
	services ports.AcceptedServiceRepository
}

func NewProviderConsumer(svc ports.AcceptedServiceRepository) *ProviderConsumer {
	return &ProviderConsumer{services: svc}
}

func (c *ProviderConsumer) Handle(msg []byte) {
	var ev events.ProviderEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		log.Println("provider_consumer: invalid payload", err)
		return
	}

	serviceOID, err := primitive.ObjectIDFromHex(ev.ServiceID)
	if err != nil {
		log.Println("invalid service id:", err)
		return
	}

	ctx := context.Background()

	switch ev.Action {

	case "assign":
		providerOID, err := primitive.ObjectIDFromHex(ev.ProviderID)
		if err != nil {
			return
		}

		_ = c.services.UpdateByID(ctx, serviceOID, bson.M{
			"$set": bson.M{
				"provider": providerOID,
				"status":   domain.StatusProviderAssigned,
				"assignedAt": time.Now(),
			},
		})

	case "release":
		_ = c.services.UpdateByID(ctx, serviceOID, bson.M{
			"$set": bson.M{
				"provider": nil,
				"status":   domain.StatusSearching,
			},
		})
	}
}
