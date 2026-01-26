package repository

import (
    "context"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "time"
)

type DeviceToken struct {
	ID        string    `bson:"_id,omitempty"`
	OwnerID  string    `bson:"ownerId"`
	OwnerType string   `bson:"ownerType"` // user | provider
	Token     string    `bson:"token"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

type DeviceTokenRepo struct {
	col *mongo.Collection
}

func NewDeviceTokenRepo(db *mongo.Database) *DeviceTokenRepo {
	return &DeviceTokenRepo{
		col: db.Collection("device_tokens"),
	}
}

// Save or update token
func (r *DeviceTokenRepo) Save(
	ctx context.Context,
	ownerID string,
	ownerType string,
	token string,
) error {

	filter := bson.M{
		"ownerId": ownerID,
		"token":   token,
	}

	update := bson.M{
		"$set": bson.M{
			"ownerId":  ownerID,
			"ownerType": ownerType,
			"token":     token,
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

// GetTokens by owner
func (r *DeviceTokenRepo) GetTokens(
	ctx context.Context,
	ownerID string,
) ([]string, error) {

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cur, err := r.col.Find(ctx2, bson.M{"ownerId": ownerID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx2)

	var tokens []string

	for cur.Next(ctx2) {
		var dt DeviceToken
		if err := cur.Decode(&dt); err == nil {
			tokens = append(tokens, dt.Token)
		}
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}
