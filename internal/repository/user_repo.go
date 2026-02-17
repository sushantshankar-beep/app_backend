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
	filter := bson.M{
		"phone": phone,
		"isActive": bson.M{"$ne": domain.PROVIDER_DELETED},
	}
	var u domain.User
	err := r.col.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
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

func (r *UserRepo) SetPrimaryVehicle( ctx context.Context, userID primitive.ObjectID, vehicleID primitive.ObjectID ) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{
			"_id": userID,
			"$or": []bson.M{
				{"primaryVehicleId": bson.M{"$exists": false}},
				{"primaryVehicleId": nil},
			},
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

func (r *UserRepo) AddFallbackVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleID primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		mongo.Pipeline{
			{
				{"$set", bson.M{
					"fallbackVehicleIds": bson.M{
						"$cond": bson.M{
							"if": bson.M{"$isArray": "$fallbackVehicleIds"},
							"then": bson.M{
								"$setUnion": []interface{}{
									"$fallbackVehicleIds",
									[]primitive.ObjectID{vehicleID},
								},
							},
							"else": []primitive.ObjectID{vehicleID},
						},
					},
				}},
			},
		},
	)

	return err
}

func (r *UserRepo) UpdateTotalExpense(
	ctx context.Context,
	userID primitive.ObjectID,
	total float64,
) error {

	_, err := r.col.UpdateByID(
		ctx,
		userID,
		bson.M{
			"$set": bson.M{
				"totalExpense": total,
				"updatedAt":    time.Now(),
			},
		},
	)

	return err
}

func (r *UserRepo) SetFallbackVehicles(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleIDs []primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"fallbackVehicleIds": vehicleIDs}},
	)

	return err
}

func (r *UserRepo) RemoveFallbackVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
	vehicleID primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$pull": bson.M{"fallbackVehicleIds": vehicleID}},
	)

	return err
}

func (r *UserRepo) ClearPrimaryVehicle(
	ctx context.Context,
	userID primitive.ObjectID,
) error {

	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$unset": bson.M{"primaryVehicleId": ""}},
	)

	return err
}

func (r *UserRepo) UpdateRating(ctx context.Context, userID domain.UserID, rating string) error {
	objID, err := primitive.ObjectIDFromHex(string(userID))
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"rating":    rating,
			"updatedAt": time.Now(),
		},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user rating: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepo) IncrementComplaintCount(ctx context.Context, userID primitive.ObjectID) error {
    filter := bson.M{"_id": userID}
    update := bson.M{"$inc": bson.M{"complaintsCount": 1}}
    _, err := r.col.UpdateOne(ctx, filter, update)
    return err
}