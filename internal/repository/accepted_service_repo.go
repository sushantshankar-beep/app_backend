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
    oid, _ := primitive.ObjectIDFromHex(string(providerID))

    filter := bson.M{
        "provider": oid,
        "paymentStatus": "paid",
    }

    opts := options.Find().
        SetSort(bson.M{"createdAt": -1}).
        SetSkip(int64(skip)).
        SetLimit(int64(limit))

    cur, err := r.col.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }

    var result []domain.AcceptedService
    if err := cur.All(ctx, &result); err != nil {
        return nil, err
    }

    return result, nil
}

func (r *AcceptedServiceRepo) FindByIDAndProvider(ctx context.Context, id string, providerID domain.ProviderID) (*domain.AcceptedService, error) {
    oid, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, domain.ErrNotFound
    }

    providerOID, _ := primitive.ObjectIDFromHex(string(providerID))

    filter := bson.M{
        "_id": oid,
        "provider": providerOID,
        "paymentStatus": "paid",
    }

    var result domain.AcceptedService
    err = r.col.FindOne(ctx, filter).Decode(&result)
    if err != nil {
        return nil, err
    }

    return &result, nil
}


func (r *AcceptedServiceRepo) FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.AcceptedService, error) {
    var service domain.AcceptedService

    err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&service)
    if err != nil {
        return nil, err
    }

    return &service, nil
}

func (r *AcceptedServiceRepo) Aggregate(ctx context.Context, pipeline mongo.Pipeline) ([]map[string]any, error) {
    cur, err := r.col.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var results []map[string]any
    if err := cur.All(ctx, &results); err != nil {
        return nil, err
    }

    return results, nil
}
