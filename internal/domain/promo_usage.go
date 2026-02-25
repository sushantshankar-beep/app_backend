package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PromoUsage struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	PromoID primitive.ObjectID `bson:"promoId" json:"promoId"`
	UserID  string             `bson:"userId" json:"userId"`
	Phone   string             `bson:"phone" json:"phone"`
	UsedAt  time.Time          `bson:"usedAt" json:"usedAt"`
}