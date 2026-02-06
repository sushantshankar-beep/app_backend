package repository

import (
	"context"
	// "time"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VehicleRepo struct {
	col *mongo.Collection
}

func NewVehicleRepo(db *mongo.Database) *VehicleRepo {
	return &VehicleRepo{col: db.Collection("vehicles")}
}

func (r *VehicleRepo) FindByNumber(ctx context.Context, number string) (*domain.Vehicle, error) {
	var v domain.Vehicle
	err := r.col.FindOne(ctx, bson.M{"vehicleNumber": number}).Decode(&v)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &v, err
}

func (r *VehicleRepo) Create(ctx context.Context, v *domain.Vehicle) error {
	res, err := r.col.InsertOne(ctx, v)
	if err != nil {
		return err
	}
	v.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}
func (r *VehicleRepo) FindByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.Vehicle, error) {

	var v domain.Vehicle
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&v)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &v, err
}
func (r *VehicleRepo) FindByIDs(
	ctx context.Context,
	ids []primitive.ObjectID,
) ([]domain.Vehicle, error) {

	cur, err := r.col.Find(
		ctx,
		bson.M{"_id": bson.M{"$in": ids}},
	)

	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var vehicles []domain.Vehicle

	if err := cur.All(ctx, &vehicles); err != nil {
		return nil, err
	}

	return vehicles, nil
}

func (r *VehicleRepo) Update(
	ctx context.Context,
	id primitive.ObjectID,
	updateData bson.M,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updateData},
	)

	return err
}


func (r *VehicleRepo) CountByUserID(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	filter := bson.M{"userId": userID}
	return r.col.CountDocuments(ctx, filter)
}