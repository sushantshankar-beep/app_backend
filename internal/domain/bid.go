package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Bid struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ServiceRequest  primitive.ObjectID `bson:"serviceRequest" json:"serviceRequest"`
	BidID           int64              `bson:"id" json:"bidId"`
	Provider        primitive.ObjectID `bson:"provider" json:"provider"`
	OfferedPrice    float64            `bson:"offeredPrice" json:"offeredPrice"`
	BasePrice       *float64           `bson:"basePrice,omitempty" json:"basePrice,omitempty"`
	EstimatedTime   EstimatedTime      `bson:"estimatedTime" json:"estimatedTime"`
	Distance        string             `bson:"distance" json:"distance"`
	Message         string             `bson:"message,omitempty" json:"message,omitempty"`
	Status          string             `bson:"status" json:"status"`
	ExpiresAt       time.Time          `bson:"expiresAt" json:"expiresAt"`
	AcceptedAt      *time.Time         `bson:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	RejectedAt      *time.Time         `bson:"rejectedAt,omitempty" json:"rejectedAt,omitempty"`
	ViewedByUser    bool               `bson:"viewedByUser" json:"viewedByUser"`
	ViewedAt        *time.Time         `bson:"viewedAt,omitempty" json:"viewedAt,omitempty"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type EstimatedTime struct {
	Value int64  `bson:"value" json:"value"`
	Unit  string `bson:"unit" json:"unit"`
}