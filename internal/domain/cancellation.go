package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CancellationLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	ServiceID   primitive.ObjectID `bson:"serviceId"`
	ProviderID  primitive.ObjectID `bson:"providerId"`
	CancelledBy string             `bson:"cancelledBy"`
	Reason      string             `bson:"reason,omitempty"`
	CreatedAt  time.Time           `bson:"createdAt"`
}
