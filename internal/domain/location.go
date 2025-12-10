package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Location struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"userId" json:"userId"`
	Latitude  float64            `bson:"latitude" json:"latitude"`
	Longitude float64            `bson:"longitude" json:"longitude"`
	Address   string             `bson:"address" json:"address"`
	City      string             `bson:"city" json:"city"`
	Pincode   string             `bson:"pincode" json:"pincode"`
}
