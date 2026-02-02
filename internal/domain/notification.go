package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`

	OwnerID   primitive.ObjectID `bson:"ownerId"`
	OwnerType string             `bson:"ownerType"`

	ServiceID primitive.ObjectID `bson:"serviceId"`

	Title string `bson:"title"`
	Body  string `bson:"body"`
	Data  map[string]string `bson:"data"`

	Read bool `bson:"read"`
	Status string `bson:"status"`

	CreatedAt time.Time `bson:"createdAt"`
}

