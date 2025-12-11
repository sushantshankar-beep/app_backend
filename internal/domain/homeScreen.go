package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HomepageID string

type Homepage struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserCohort string             `bson:"userCohort" json:"userCohort"`
	Location   LocationInfo       `bson:"location" json:"location"`
	Banners    []Banner           `bson:"banners" json:"banners"`
	Categories []Category         `bson:"categories" json:"categories"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
	IsActive   bool               `bson:"isActive" json:"isActive"`
}

type LocationInfo struct {
	City             string  `bson:"city" json:"city"`
	Address          string  `bson:"address" json:"address"`
	Latitude         float64 `bson:"latitude" json:"latitude"`
	Longitude        float64 `bson:"longitude" json:"longitude"`
	Pincode          string  `bson:"pincode" json:"pincode"`
	FormattedAddress string  `bson:"formattedAddress" json:"formattedAddress"`
}

type Banner struct {
	Title             string         `bson:"title" json:"title"`
	ImageURL          string         `bson:"imageUrl" json:"imageUrl"`
	RedirectionURL    string         `bson:"redirectionUrl" json:"redirectionUrl"`
	RedirectionParams map[string]any `bson:"redirectionParams" json:"redirectionParams"`
}

type Category struct {
	Name           string `bson:"name" json:"name"`
	IconURL        string `bson:"iconUrl" json:"iconUrl"`
	RedirectionURL string `bson:"redirectionUrl" json:"redirectionUrl"`
}