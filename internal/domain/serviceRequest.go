package domain

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServiceRequest struct {
	ID              primitive.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	User            primitive.ObjectID   `bson:"user" json:"user"`
	ServiceID       int64                `bson:"id" json:"serviceId"`
	VehicleNumber   string               `bson:"vehicleNumber" json:"vehicleNumber"`
	VehicleType     string               `bson:"vehicleType" json:"vehicleType"`
	Brand           string               `bson:"brand" json:"brand"`
	Model           string               `bson:"model" json:"model"`
	BasePrice       float64              `bson:"basePrice" json:"basePrice"`
	FinalPrice      *float64             `bson:"finalPrice,omitempty" json:"finalPrice,omitempty"`
	Year            *int                 `bson:"year,omitempty" json:"year,omitempty"`
	FuelType        string               `bson:"fuelType" json:"fuelType"`
	ServiceType     string               `bson:"serviceType" json:"serviceType"`
	ServiceBidType  string               `bson:"serviceBidType" json:"serviceBidType"`
	Problems        []string             `bson:"problems,omitempty" json:"problems,omitempty"`
	Description     string               `bson:"description,omitempty" json:"description,omitempty"`
	Location        *Location            `bson:"location,omitempty" json:"location,omitempty"`
	Address         string               `bson:"address,omitempty" json:"address,omitempty"`
	ScheduledDate   *time.Time           `bson:"scheduledDate,omitempty" json:"scheduledDate,omitempty"`
	Status          string               `bson:"status" json:"status"`
	AcceptedBid     *primitive.ObjectID  `bson:"acceptedBid,omitempty" json:"acceptedBid,omitempty"`
	TotalBids       int                  `bson:"totalBids" json:"totalBids"`
	ExpiresAt       time.Time            `bson:"expiresAt" json:"expiresAt"`
	Metadata        ServiceMetadata      `bson:"metadata" json:"metadata"`
	CreatedAt       time.Time            `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time            `bson:"updatedAt" json:"updatedAt"`
}

type GeoJSONLocation struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"`
}

type ServiceMetadata struct {
	BroadcastedTo int        `bson:"broadcastedTo" json:"broadcastedTo"`
	LastBidAt     *time.Time `bson:"lastBidAt,omitempty" json:"lastBidAt,omitempty"`
}