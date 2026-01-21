package repository

import (
	"context"

	"app_backend/internal/domain"
	"time"
	// "errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepo struct {
	col *mongo.Collection
}
func (r *UserRepo) Col() *mongo.Collection {
	return r.col
}
func (r *UserRepo) UpdateVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleID primitive.ObjectID,
) error {

	_, err := r.col.UpdateByID(ctx, userID, bson.M{
		"$set": bson.M{
			"vehicleId": vehicleID,
			"updatedAt": time.Now(),
		},
	})
	return err
}




func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{col: db.Collection("users")}
}
func (r *UserRepo) AddComplaint(ctx context.Context, userID primitive.ObjectID, complaintID primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, userID, bson.M{
		"$push": bson.M{"complaintsSubmitted": complaintID},
	})
	return err
}

func (r *UserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var u domain.User
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	 res, err := r.col.InsertOne(ctx, u)
    if err != nil {
        return err
    }
    u.ID = domain.UserID(res.InsertedID.(primitive.ObjectID).Hex())
	return nil
}
func (r *UserRepo) GetByID(ctx context.Context,userID primitive.ObjectID,) (*domain.User, error) {
	var u domain.User
	fmt.Println(userID)
	err := r.col.FindOne(ctx, bson.M{"_id": userID}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	fmt.Println(&u)
	return &u, err
}
func (r *UserRepo) UpdateByID(ctx context.Context,id primitive.ObjectID,update bson.M) (*domain.User, error) {
	_, err := r.col.UpdateByID(ctx, id, bson.M{"$set": update})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}
func (r *UserRepo) UpdateRaw(
	ctx context.Context,
	userID primitive.ObjectID,
	update bson.M,
) error {
	_, err := r.col.UpdateByID(ctx, userID, update)
	return err
}
func (r *UserRepo) SetPrimaryVehicle(
		ctx context.Context,
		userID primitive.ObjectID,
		vehicleID primitive.ObjectID,
	) error {

		_, err := r.col.UpdateOne(
			ctx,
			bson.M{
				"_id": userID,
				"primaryVehicleId": primitive.NilObjectID,
			},
			bson.M{
				"$set": bson.M{"primaryVehicleId": vehicleID},
			},
		)

		return err
	}
func (r *UserRepo) FindByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.User, error) {

	var user domain.User
	if err := r.col.FindOne(
		ctx,
		bson.M{"_id": id},
	).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}


func (r *UserRepo) AddFallbackVehicle(ctx context.Context,userID primitive.ObjectID,vehicleID primitive.ObjectID) error {
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{
			"$addToSet": bson.M{
				"fallbackVehicleIds": vehicleID,
			},
		},
	)

	return err
}

