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
   FIND BY OWNER (USER / PROVIDER)
============================================================ */

func (r *NotificationRepo) FindByOwner(
	ctx context.Context,
	ownerID primitive.ObjectID,
	ownerType string,
	limit int64,
	skip int64,
) ([]domain.Notification, error) {

	filter := bson.M{
		"ownerId":   ownerID,
		"ownerType": ownerType,
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

/* ============================================================
   COUNT UNREAD
============================================================ */

func (r *NotificationRepo) CountUnread(
	ctx context.Context,
	ownerID primitive.ObjectID,
	ownerType string,
) (int64, error) {

	return r.col.CountDocuments(ctx, bson.M{
		"ownerId":   ownerID,
		"ownerType": ownerType,
		"read":      false,
	})
}

/* ============================================================
   MARK SINGLE READ
============================================================ */

func (r *NotificationRepo) MarkRead(
	ctx context.Context,
	id primitive.ObjectID,
	ownerID primitive.ObjectID,
) error {

	res, err := r.col.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"ownerId": ownerID,
		},
		bson.M{
			"$set": bson.M{
				"read": true,
			},
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

/* ============================================================
   MARK ALL READ
============================================================ */

func (r *NotificationRepo) MarkAllRead(
	ctx context.Context,
	ownerID primitive.ObjectID,
	ownerType string,
) error {

	_, err := r.col.UpdateMany(
		ctx,
		bson.M{
			"ownerId":   ownerID,
			"ownerType": ownerType,
			"read":      false,
		},
		bson.M{
			"$set": bson.M{
				"read": true,
			},
		},
	)

	return err
}

/* ============================================================
   UPDATE STATUS
============================================================ */

func (r *NotificationRepo) UpdateStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status string,
) error {

	_, err := r.col.UpdateByID(ctx, id, bson.M{
		"$set": bson.M{
			"status": status,
		},
	})

	return err
}
