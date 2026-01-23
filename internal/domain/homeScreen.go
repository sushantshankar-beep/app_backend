package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Homepage struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	Scope HomepageScope `bson:"scope" json:"scope"`

	Header HeaderSection `bson:"header" json:"header"`
	Banners []Banner     `bson:"banners" json:"banners"`
	Categories []Category `bson:"categories" json:"categories"`
	BottomNav []BottomNavItem `bson:"bottomNav" json:"bottomNav"`

	Priority int  `bson:"priority" json:"priority"`
	IsActive string `bson:"isActive" json:"isActive"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type HomepageScope struct {
	Country string `bson:"country,omitempty" json:"country,omitempty"`
	State   string `bson:"state,omitempty" json:"state,omitempty"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
}

type HeaderSection struct {
	Title    string `bson:"title" json:"title"`
	Subtitle string `bson:"subtitle" json:"subtitle"`
}

type Banner struct {
	Title          string         `bson:"title" json:"title"`
	ImageURL       string         `bson:"imageUrl" json:"imageUrl"`
	RedirectURL    string         `bson:"redirectUrl" json:"redirectUrl"`
	RedirectParams map[string]any `bson:"redirectParams" json:"redirectParams"`
}

type Category struct {
	Title       string `bson:"title" json:"title"`
	IconURL     string `bson:"iconUrl" json:"iconUrl"`
	RedirectURL string `bson:"redirectUrl" json:"redirectUrl"`
}

type BottomNavItem struct {
	Title string `bson:"title" json:"title"`
	Icon  string `bson:"icon" json:"icon"`
	Route string `bson:"route" json:"route"`
}
