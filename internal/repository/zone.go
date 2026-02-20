package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ZoneRepository struct {
	collection *mongo.Collection
}

func NewZoneRepository(col *mongo.Collection) *ZoneRepository {
	return &ZoneRepository{
		collection: col,
	}
}

func (r *ZoneRepository) FindActiveByCity(
	ctx context.Context,
	city string,
) (*domain.Zone, error) {

	filter := bson.M{
		"zoneName": bson.M{
			"$regex":   "^" + city + "$",
			"$options": "i",
		},
		"isActive": true,
	}

	var zone domain.Zone
	err := r.collection.FindOne(ctx, filter).Decode(&zone)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &zone, nil
}

