package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HomepageRepo struct {
	col *mongo.Collection
}

func NewHomepageRepo(db *mongo.Database) *HomepageRepo {
	return &HomepageRepo{col: db.Collection("homepages")}
}

func (r *HomepageRepo) Create(ctx context.Context, h *domain.Homepage) error {
	res, err := r.col.InsertOne(ctx, h)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		h.ID = oid
	}
	return nil
}

func (r *HomepageRepo) Update(ctx context.Context, h *domain.Homepage) error {
	update := bson.M{
		"$set": bson.M{
			"userCohort": h.UserCohort,
			"location":   h.Location,
			"banners":    h.Banners,
			"categories": h.Categories,
			"isActive":   h.IsActive,
			"updatedAt":  h.UpdatedAt,
		},
	}

	_, err := r.col.UpdateByID(ctx, h.ID, update)
	return err
}

func (r *HomepageRepo) FindByID(ctx context.Context, id string) (*domain.Homepage, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var homepage domain.Homepage
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&homepage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &homepage, nil
}

func (r *HomepageRepo) FindAll(ctx context.Context, skip, limit int) ([]domain.Homepage, error) {
	cursor, err := r.col.Find(ctx, bson.M{}, &options.FindOptions{
		Skip:  &[]int64{int64(skip)}[0],
		Limit: &[]int64{int64(limit)}[0],
		Sort:  bson.M{"createdAt": -1},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var homepages []domain.Homepage
	if err = cursor.All(ctx, &homepages); err != nil {
		return nil, err
	}

	return homepages, nil
}

func (r *HomepageRepo) Count(ctx context.Context, filter bson.M) (int64, error) {
	return r.col.CountDocuments(ctx, filter)
}