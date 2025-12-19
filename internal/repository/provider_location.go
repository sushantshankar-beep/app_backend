package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (r *ProviderRepo) UpdateLocation(
	ctx context.Context,
	providerID primitive.ObjectID,
	lat, lon float64,
) error {

	_, err := r.col.UpdateByID(
		ctx,
		providerID,
		bson.M{
			"$set": bson.M{
				"providerLocation": bson.M{
					"lat": lat,
					"lng": lon,
				},
				"updatedAt": time.Now(),
			},
		},
	)

	return err
}
