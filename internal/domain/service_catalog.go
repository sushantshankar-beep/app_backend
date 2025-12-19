package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type ServiceCatalog struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name             string             `bson:"name" json:"name"`
	BasePrice        float64            `bson:"basePrice" json:"basePrice"`
	MaxBiddingAmount float64            `bson:"maxBiddingAmount" json:"maxBiddingAmount"`
	GSTPercent       float64            `bson:"gstPercent" json:"gstPercent"`
	Discount         float64            `bson:"discount" json:"discount"`
	CouponAmount     float64            `bson:"couponAmount" json:"couponAmount"`
}
