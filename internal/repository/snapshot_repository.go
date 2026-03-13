package repository

import (
	"context"

	"app_backend/internal/domain"
"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "fmt"
)

type SnapshotRepo struct {
	col *mongo.Collection
}

func NewSnapshotRepo(db *mongo.Database) *SnapshotRepo {
	return &SnapshotRepo{
		col: db.Collection("accepted_service_snapshots"),
	}
}

func (r *SnapshotRepo) GetByServiceID(ctx context.Context, serviceID primitive.ObjectID) (*domain.ActiveServiceSnapshot, error) {
    var snap domain.ActiveServiceSnapshot

    opts := options.FindOne().SetSort(bson.D{{Key: "snapshotAt", Value: -1}})

    err := r.col.FindOne(ctx, bson.M{"serviceId": serviceID}, opts).Decode(&snap)
    if err != nil {
        return nil, err
    }
    return &snap, nil
}

func (r *SnapshotRepo) GetByProviderAndStatus(
	ctx context.Context,
	providerID primitive.ObjectID,
	statuses []domain.ServiceStatus,
) ([]domain.ActiveServiceSnapshot, error) {
	vals := make([]string, 0, len(statuses))
	for _, s := range statuses {
		vals = append(vals, string(s))
	}

	filter := bson.M{
		"service.provider": providerID,
		"service.status": bson.M{
			"$in": vals,
		},
	}

	cur, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var snaps []domain.ActiveServiceSnapshot
	if err := cur.All(ctx, &snaps); err != nil {
		return nil, err
	}

	return snaps, nil
}

func (r *SnapshotRepo) GetByServiceIDAndProvider(
    ctx context.Context, 
    serviceID primitive.ObjectID, 
    providerID string,
) (*domain.ActiveServiceSnapshot, error) {
    var snap domain.ActiveServiceSnapshot

    filter := bson.M{
        "serviceId":                    serviceID,
        "$or": []bson.M{
            {"service.provider":            providerID},
            {"service.cancelledProviderID": providerID},
        },
    }

    opts := options.FindOne().SetSort(bson.D{{Key: "snapshotAt", Value: -1}})
    err := r.col.FindOne(ctx, filter, opts).Decode(&snap)
    if err != nil {
        return nil, err
    }
    return &snap, nil
}