package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

/* ============================================================
   MODEL
============================================================ */

type DeviceToken struct {
	ID        string    `bson:"_id,omitempty"`
	OwnerID   string    `bson:"ownerId"`
	OwnerType string    `bson:"ownerType"` // user | provider
	Token     string    `bson:"token"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

/* ============================================================
   REPO
============================================================ */

type DeviceTokenRepo struct {
	col *mongo.Collection
}

func NewDeviceTokenRepo(db *mongo.Database) *DeviceTokenRepo {
	return &DeviceTokenRepo{
		col: db.Collection("device_tokens"),
	}
}

/* ============================================================
   SAVE / UPSERT (ROLE SAFE)
============================================================ */

func (r *DeviceTokenRepo) Save(
	ctx context.Context,
	ownerID string,
	ownerType string,
	token string,
) error {

	filter := bson.M{
		"token":     token,
		"ownerId":   ownerID,
		"ownerType": ownerType,
	}

	update := bson.M{
		"$set": bson.M{
			"token":     token,
			"ownerId":   ownerID,
			"ownerType": ownerType,
			"updatedAt": time.Now(),
		},
	}

	_, err := r.col.UpdateOne(
		ctx,
		filter,
		update,
		options.Update().SetUpsert(true),
	)

	return err
}

/* ============================================================
   GET TOKENS (DEDUP SAFE)
============================================================ */

func (r *DeviceTokenRepo) GetTokens(
	ctx context.Context,
	ownerID string,
	ownerType string,
) ([]string, error) {

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 🔽 Add sort option
	findOptions := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.col.Find(
		ctx2,
		bson.M{
			"ownerId":   ownerID,
			"ownerType": ownerType,
		},
		findOptions,
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx2)

	seen := make(map[string]struct{})
	var tokens []string

	for cur.Next(ctx2) {
		var dt DeviceToken
		if err := cur.Decode(&dt); err == nil {
			if _, ok := seen[dt.Token]; ok {
				continue
			}

			seen[dt.Token] = struct{}{}
			tokens = append(tokens, dt.Token)
		}
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}
func (r *DeviceTokenRepo) DeleteToken(
	ctx context.Context,
	token string,
) error {

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.col.DeleteOne(
		ctx2,
		bson.M{"token": token},
	)
	if err != nil {
		return err
	}

	if res.DeletedCount > 0 {
		log.Println("Token removed:", token)
	}

	return nil
}