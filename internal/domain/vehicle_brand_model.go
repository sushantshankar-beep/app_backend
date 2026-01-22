package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Variant struct {
	Years    []int  `bson:"years" json:"years"`
	FuelType string `bson:"fuelType" json:"fuelType"`
}

type Model struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Model    string             `bson:"model" json:"model"`
	Segment  string             `bson:"segment" json:"segment"`
	Variants []Variant          `bson:"variants" json:"variants"`
}

type Brand struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Make        string             `bson:"make" json:"make"`
	VehicleType string             `bson:"vehicleType" json:"vehicleType"`
	Models      []Model            `bson:"models" json:"models"`
}
