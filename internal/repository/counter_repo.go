package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CounterRepo struct {
	col *mongo.Collection
}

func NewCounterRepo(db *mongo.Database) *CounterRepo {
	return &CounterRepo{
		col: db.Collection("counters"),
	}
}

func (r *CounterRepo) Next(ctx context.Context, name string) (int64, error) {
	var res struct {
		Value int64 `bson:"value"`
	}
	err := r.col.FindOneAndUpdate(ctx,bson.M{"_id": name},bson.M{"$inc": bson.M{"value": 1}}).Decode(&res)
	if err == mongo.ErrNoDocuments {
		_, err = r.col.InsertOne(ctx, bson.M{
			"_id":   name,
			"value": 1,
		})
		return 1, err
	}

	return res.Value, err
}
