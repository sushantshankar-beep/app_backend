package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RatingType string

const (
	RATING_USER     RatingType = "user"
	RATING_PROVIDER RatingType = "provider"
)

type Rating struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BookingID       primitive.ObjectID `bson:"bookingId" json:"bookingId"`
	RaterID         primitive.ObjectID `bson:"raterId" json:"raterId"`
	RateeID         primitive.ObjectID `bson:"rateeId" json:"rateeId"`
	RatingType      RatingType         `bson:"ratingType" json:"ratingType"`
	Stars           int                `bson:"stars" json:"stars"`
	Review          string             `bson:"review" json:"review"`
	Recommended     *bool               `bson:"recommended" json:"recommended"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}