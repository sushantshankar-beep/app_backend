package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Zone struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ZoneName  string             `bson:"zoneName"`
	StateName string             `bson:"stateName"`
	IsActive  bool               `bson:"isActive"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}
