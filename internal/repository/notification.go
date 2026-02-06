package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/* ============================================================
   REPO
============================================================ */

type NotificationRepo struct {
	col *mongo.Collection
}

func NewNotificationRepo(db *mongo.Database) *NotificationRepo {
	return &NotificationRepo{
		col: db.Collection("notifications"),
	}
}

/* ============================================================
   CREATE
============================================================ */

func (r *NotificationRepo) Create(
	ctx context.Context,
	n *domain.Notification,
) error {

	res, err := r.col.InsertOne(ctx, n)
	if err != nil {
		return err
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		n.ID = oid
	}

	return nil
}

/* ============================================================
   FIND LATEST SERVICE FOR OWNER  ✅ NEW
============================================================ */

func (r *NotificationRepo) FindLatestServiceForOwner(
	ctx context.Context,
	ownerID primitive.ObjectID,
	ownerType string,
) (primitive.ObjectID, error) {

	pipeline := mongo.Pipeline{

		{{"$match", bson.M{
			"ownerId":   ownerID,
			"ownerType": ownerType,
		}}},

		{{"$sort", bson.M{"createdAt": -1}}},

		{{"$group", bson.M{
			"_id": "$serviceId",
			"latest": bson.M{"$first": "$createdAt"},
		}}},

		{{"$sort", bson.M{"latest": -1}}},

		{{"$limit", 1}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return primitive.NilObjectID, err
	}
	defer cur.Close(ctx)

	var out []struct {
		ID primitive.ObjectID `bson:"_id"`
	}

	if err := cur.All(ctx, &out); err != nil {
		return primitive.NilObjectID, err
	}

	if len(out) == 0 {
		return primitive.NilObjectID, mongo.ErrNoDocuments
	}

	return out[0].ID, nil
}

/* ============================================================
   FIND BY SERVICE
============================================================ */

func (r *NotificationRepo) FindByService(
	ctx context.Context,
	ownerID primitive.ObjectID,
	ownerType string,
	serviceID primitive.ObjectID,
	limit int64,
	skip int64,
) ([]domain.Notification, error) {

	filter := bson.M{
		"ownerId":   ownerID,
		"ownerType": ownerType,
		"serviceId": serviceID,
		"title": bson.M{
			"$in": []string{
				"Service Completed",
				"Booking Cancelled By User",
				"Booking Cancelled By Provider",
				"Booking Cancelled",
				"Payment Completed",
			},
		},
	}
	opts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetLimit(limit).
		SetSkip(skip)

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var res []domain.Notification

	if err := cur.All(ctx, &res); err != nil {
		return nil, err
	}

	return res, nil
}
