package repository

import (
	"context"
	"time"
    "errors"
	"app_backend/internal/domain"
     "go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type PromoRepo struct {
	col *mongo.Collection
}

func NewPromoRepo(db *mongo.Database) *PromoRepo {
	return &PromoRepo{col: db.Collection("promoCodes")}
}

func (r *PromoRepo) GetByCode(ctx context.Context, code string) (*domain.PromoCode, error) {
	var promo domain.PromoCode
	err := r.col.FindOne(ctx, bson.M{"code": code}).Decode(&promo)
	if err != nil {
		return nil, err
	}
	return &promo, nil
}

func (r *PromoRepo) IncrementUsage(ctx context.Context, promoID primitive.ObjectID) error {

	filter := bson.M{
		"_id": promoID,
		"$or": []bson.M{
			{"globalRedemptionCap": 0}, // unlimited
			{
				"$expr": bson.M{
					"$lt": []interface{}{"$usageCount", "$globalRedemptionCap"},
				},
			},
		},
	}

	update := bson.M{
		"$inc": bson.M{"usageCount": 1},
		"$set": bson.M{"updatedAt": time.Now()},
	}

	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return errors.New("promo redemption limit reached")
	}

	return nil
}

func (r *PromoRepo) GetActiveForService(
	ctx context.Context,
	serviceType string,
	search string,
	page, limit int,
) ([]*domain.PromoCode, int64, error) {

	now := time.Now().UTC()

	filter := bson.M{
		"status": "active",
		"startAt": bson.M{"$lte": now},
		"$or": []bson.M{
			{"endAt": nil},
			{"endAt": bson.M{"$gte": now}},
		},
	}

	if search != "" {
		regex := bson.M{"$regex": search, "$options": "i"}
		filter["$and"] = []bson.M{
			{"$or": []bson.M{
				{"code": regex},
				{"title": regex},
			}},
		}
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := (page - 1) * limit

	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit)).
		SetSort(bson.M{"createdAt": -1})

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var promos []*domain.PromoCode
	if err := cursor.All(ctx, &promos); err != nil {
		return nil, 0, err
	}

	return promos, total, nil
}
