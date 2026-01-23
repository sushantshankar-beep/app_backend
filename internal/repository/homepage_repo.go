package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HomepageRepo struct {
	col *mongo.Collection
}

func NewHomepageRepo(db *mongo.Database) *HomepageRepo {
	return &HomepageRepo{
		col: db.Collection("homepages"),
	}
}

func (r *HomepageRepo) FindByLocation(
	ctx context.Context,
	country, state, city string,
) (*domain.Homepage, error) {

	filter := bson.M{
		"isActive": "active",
		"$or": []bson.M{
			{"scope.city": city},
			{"scope.state": state},
			{"scope.country": country},
			{"scope": bson.M{}}, // fallback
		},
	}

	opts := options.FindOne().
		SetSort(bson.D{{Key: "priority", Value: -1}})

	var homepage domain.Homepage
	err := r.col.FindOne(ctx, filter, opts).Decode(&homepage)
	if err != nil {
		return nil, err
	}

	return &homepage, nil
}