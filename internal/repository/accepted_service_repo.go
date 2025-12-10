// repository/accepted_service.go
package repository

import (
    "app_backend/internal/domain"
    "context"


    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type AcceptedServiceRepo struct {
    col *mongo.Collection
}

func NewAcceptedServiceRepo(db *mongo.Database) *AcceptedServiceRepo {
    return &AcceptedServiceRepo{col: db.Collection("acceptedservices")}
}

func (r *AcceptedServiceRepo) Find(ctx context.Context, filter bson.M, skip, limit int) ([]domain.AcceptedService, error) {
    opts := options.Find().
        SetSkip(int64(skip)).
        SetLimit(int64(limit)).
        SetSort(bson.M{"createdAt": -1})

    cur, err := r.col.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var results []domain.AcceptedService
    if err := cur.All(ctx, &results); err != nil {
        return nil, err
    }

    return results, nil
}

func (r *AcceptedServiceRepo) Count(ctx context.Context, filter bson.M) (int64, error) {
    return r.col.CountDocuments(ctx, filter)
}

func (r *AcceptedServiceRepo) ListByProvider(ctx context.Context, providerID domain.ProviderID, skip, limit int) ([]domain.AcceptedService, error) {
    filter := bson.M{
        "provider": providerID,
        "paymentStatus": "paid",
    }

    opts := options.Find().
        SetSkip(int64(skip)).
        SetLimit(int64(limit)).
        SetSort(bson.M{"createdAt": -1})

    cur, err := r.col.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var services []domain.AcceptedService
    if err := cur.All(ctx, &services); err != nil {
        return nil, err
    }

    return services, nil
}

func (r *AcceptedServiceRepo) FindByIDAndProvider(ctx context.Context, id string, providerID domain.ProviderID) (*domain.AcceptedService, error) {
    objID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }

    filter := bson.M{
        "_id": objID,
        "provider": providerID,
        "paymentStatus": "paid",
    }

    var svc domain.AcceptedService
    if err := r.col.FindOne(ctx, filter).Decode(&svc); err != nil {
        return nil, err
    }

    return &svc, nil
}