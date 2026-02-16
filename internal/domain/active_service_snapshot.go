package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActiveServiceSnapshot struct {
	ID         primitive.ObjectID `bson:"_id"`
	ServiceID primitive.ObjectID `bson:"serviceId"`
	SnapshotAt time.Time         `bson:"snapshotAt"`
	Service    AcceptedService   `bson:"service"`
}
