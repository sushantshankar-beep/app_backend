package repository

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AcceptedServiceRepo struct {
	col *mongo.Collection
}

func NewAcceptedServiceRepo(db *mongo.Database) *AcceptedServiceRepo {
	col := db.Collection("acceptedservices")

	// 🔥 IMPORTANT INDEXES (RUN ONCE)
	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys: bson.M{"provider": 1, "paymentStatus": 1, "createdAt": -1},
		},
		{
			Keys: bson.M{"user": 1, "createdAt": -1},
		},
	})

	return &AcceptedServiceRepo{col: col}
}

func (r *AcceptedServiceRepo) Col() *mongo.Collection {
	return r.col
}

/*
CREATE SERVICE
*/
func (r *AcceptedServiceRepo) Create(
	ctx context.Context,
	svc *domain.AcceptedService,
) error {

	res, err := r.col.InsertOne(ctx, svc)
	if err != nil {
		return err
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return fmt.Errorf("insertedID is not ObjectID")
	}

	svc.ID = oid // 🔥 THIS LINE IS CRITICAL
	return nil
}



/*
LIST SERVICES (GENERIC)
*/
func (r *AcceptedServiceRepo) Find(ctx context.Context,filter bson.M,skip, limit int) ([]domain.AcceptedService, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := options.Find().SetSkip(int64(skip)).SetLimit(int64(limit)).SetSort(bson.M{"createdAt": -1})
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

/*
COUNT
*/
func (r *AcceptedServiceRepo) Count(ctx context.Context,filter bson.M) (int64, error) {

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return r.col.CountDocuments(ctx, filter)
}

/*
LIST BY PROVIDER (SAFE)
*/
func (r *AcceptedServiceRepo) ListByProvider(
	ctx context.Context,
	providerID domain.ProviderID,
	skip, limit int,
) ([]domain.AcceptedService, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	providerOID, err := primitive.ObjectIDFromHex(string(providerID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"provider":      providerOID,
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
func (r *AcceptedServiceRepo) FindByID(
	ctx context.Context,
	id string,
) (*domain.AcceptedService, error) {

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := r.col.FindOne(
		ctx,
		bson.M{"_id": oid},
	).Decode(&svc); err != nil {
		return nil, err
	}

	return &svc, nil
}


/*
FIND BY ID + PROVIDER
*/
func (r *AcceptedServiceRepo) FindByIDAndProvider(
	ctx context.Context,
	serviceID string,
	providerID domain.ProviderID,
) (*domain.AcceptedService, error) {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	providerOID, err := primitive.ObjectIDFromHex(string(providerID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"_id":     serviceOID,
		"provider": providerOID,
	}

	var svc domain.AcceptedService
	if err := r.col.FindOne(ctx, filter).Decode(&svc); err != nil {
		return nil, err
	}

	return &svc, nil
}
func (r *AcceptedServiceRepo) GetByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.AcceptedService, error) {

	var svc domain.AcceptedService
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&svc)
	return &svc, err
}

func (r *AcceptedServiceRepo) UpdateStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status domain.ServiceStatus,
	fields bson.M,
) error {

	update := bson.M{
		"$set": bson.M{
			"status":    status,
			"updatedAt": time.Now(),
		},
	}
	for k, v := range fields {
		update["$set"].(bson.M)[k] = v
	}

	_, err := r.col.UpdateByID(ctx, id, update)
	return err
}

func (r *AcceptedServiceRepo) UpdateByID(
	ctx context.Context,
	id primitive.ObjectID,
	update bson.M,
) error {
	_, err := r.col.UpdateByID(ctx, id, update)
	return err
}

func (r *AcceptedServiceRepo) UpdatePaymentStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status domain.PaymentStatus,
) error {
	_, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"paymentStatus": status,
			"updatedAt":     time.Now(),
			"status":"confirmed",
		},
	})
	return err
}

func (r *AcceptedServiceRepo) GetBookingsByUserAndStatus(ctx context.Context, userID string, status domain.ServiceStatus) ([]domain.AcceptedService, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
if err != nil {
    return nil, fmt.Errorf("invalid userID: %w", err)
}

filter := bson.M{
    "user": objID,
    "status": status,
}
    
	var bookings []domain.AcceptedService
	cursor, err := r.col.Find(ctx, filter, &options.FindOptions{
		Sort: bson.M{"createdAt": -1},
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, err
	}

	return bookings, nil
}


func (r *AcceptedServiceRepo) GetBookingsByProviderAndStatus(ctx context.Context, userID string, status domain.ServiceStatus) ([]domain.AcceptedService, error) {
	objID, err := primitive.ObjectIDFromHex(userID)
    if err != nil {
       return nil, fmt.Errorf("invalid userID: %w", err)
    }

   filter := bson.M{
     "provider": objID,
     "status": status,
    }
    
	var bookings []domain.AcceptedService
	cursor, err := r.col.Find(ctx, filter, &options.FindOptions{
		Sort: bson.M{"createdAt": -1},
	})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, err
	}

	return bookings, nil
}

func (r *AcceptedServiceRepo) UpdateComplaintByUser(
	ctx context.Context,
	acceptedServiceId primitive.ObjectID,
	complaintId primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": acceptedServiceId},
		bson.M{
			"$set": bson.M{
				"complaintUser": complaintId,
				"updatedAt":     time.Now(),
			},
		},
	)
	return err
}

func (r *AcceptedServiceRepo) UpdateComplaintByProvider(
	ctx context.Context,
	acceptedServiceId primitive.ObjectID,
	complaintId primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": acceptedServiceId},
		bson.M{
			"$set": bson.M{
				"complaintProvider": complaintId,
				"updatedAt":         time.Now(),
			},
		},
	)
	return err
}
