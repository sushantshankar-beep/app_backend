package domain

import "time"

type ProviderID string

type Provider struct {
	ID        ProviderID `bson:"_id,omitempty" json:"id"`
	Phone     string     `bson:"phone"`
	Services  []string   `bson:"services"`
	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
}
