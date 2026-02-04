package repository

import (
	"context"
	"time"
	"fmt"
    "app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RatingRepo struct {
	collection *mongo.Collection
}

func NewRatingRepository(db *mongo.Database) *RatingRepo {
	return &RatingRepo{
		collection: db.Collection("ratings"),
	}
}

func (r *RatingRepo) Create(ctx context.Context, rating *domain.Rating) error {
	rating.ID = primitive.NewObjectID()
	rating.CreatedAt = time.Now()
	rating.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, rating)
	return err
}

func (r *RatingRepo) GetByBookingAndRater(ctx context.Context, bookingID, raterID primitive.ObjectID) (*domain.Rating, error) {
	var rating domain.Rating
	filter := bson.M{
		"bookingId": bookingID,
		"raterId":   raterID,
	}

	err := r.collection.FindOne(ctx, filter).Decode(&rating)
	if err != nil {
		return nil, err
	}

	return &rating, nil
}

func (r *RatingRepo) GetRatingsByRatee(ctx context.Context, rateeID primitive.ObjectID, ratingType domain.RatingType) ([]domain.Rating, error) {
	filter := bson.M{
		"rateeId":    rateeID,
		"ratingType": ratingType,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ratings []domain.Rating
	if err := cursor.All(ctx, &ratings); err != nil {
		return nil, err
	}

	return ratings, nil
}

func (r *RatingRepo) GetByRateeID(ctx context.Context, rateeID primitive.ObjectID) ([]*domain.Rating, error) {
	filter := bson.M{
		"rateeId": rateeID,
		"ratingType": domain.RATING_PROVIDER,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find ratings: %w", err)
	}
	defer cursor.Close(ctx)

	var ratings []*domain.Rating
	if err := cursor.All(ctx, &ratings); err != nil {
		return nil, fmt.Errorf("failed to decode ratings: %w", err)
	}

	return ratings, nil
}

func (r *RatingRepo) GetUserRatingsByRateeID(ctx context.Context, rateeID primitive.ObjectID) ([]*domain.Rating, error) {
	filter := bson.M{
		"rateeId":    rateeID,
		"ratingType": domain.RATING_USER,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find user ratings: %w", err)
	}
	defer cursor.Close(ctx)

	var ratings []*domain.Rating
	if err := cursor.All(ctx, &ratings); err != nil {
		return nil, fmt.Errorf("failed to decode user ratings: %w", err)
	}

	return ratings, nil
}
