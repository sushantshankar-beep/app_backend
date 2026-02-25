package repository

import (
    "context"
    "app_backend/internal/domain"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

type PromoUsageRepo struct {
    col *mongo.Collection
}

func NewPromoUsageRepo(db *mongo.Database) *PromoUsageRepo {
    return &PromoUsageRepo{
        col: db.Collection("promoUse"),
    }
}

func (r *PromoUsageRepo) CountByUser(ctx context.Context, promoID primitive.ObjectID, userID string) (int, error) {
    count, err := r.col.CountDocuments(ctx, bson.M{
        "promoId": promoID,
        "userId":  userID,
    })
    return int(count), err
}

func (r *PromoUsageRepo) Insert(ctx context.Context, usage domain.PromoUsage) error {
    usage.ID = primitive.NewObjectID()
    _, err := r.col.InsertOne(ctx, usage)
    return err
}