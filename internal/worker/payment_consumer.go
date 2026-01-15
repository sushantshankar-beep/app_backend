package worker

import (
	"context"
	"encoding/json"
	"log"

	"app_backend/internal/domain"
	"app_backend/internal/events"
	"app_backend/internal/ports"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentConsumer struct {
	services ports.AcceptedServiceRepository
}

func NewPaymentConsumer(svc ports.AcceptedServiceRepository) *PaymentConsumer {
	return &PaymentConsumer{services: svc}
}

func (c *PaymentConsumer) Handle(msg []byte) {
	var ev events.PaymentEvent
	if err := json.Unmarshal(msg, &ev); err != nil {
		log.Println("payment_consumer: invalid payload", err)
		return
	}

	serviceOID, err := primitive.ObjectIDFromHex(ev.ServiceID)
	if err != nil {
		log.Println("invalid service id:", err)
		return
	}

	ctx := context.Background()

	switch ev.Status {

	case string(domain.PaymentPaid):
		_ = c.services.UpdatePaymentStatus(
			ctx,
			serviceOID,
			domain.PaymentPaid,
		)

	case "grace":
		_ = c.services.UpdatePaymentStatus(
			ctx,
			serviceOID,
			domain.PaymentFailedGrace,
		)

	case "cancelled":
		_ = c.services.UpdateByID(ctx, serviceOID, bson.M{
			"$set": bson.M{
				"paymentStatus": domain.PaymentFailed,
				"status":        domain.StatusCancelled,
			},
		})
	}
}
