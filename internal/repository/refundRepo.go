package repository

import (
	"context"
	// "time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RefundRepo struct {
	col *mongo.Collection
}

func NewRefundRepo(db *mongo.Database) *RefundRepo {
	return &RefundRepo{
		col: db.Collection("refund_transactions"),
	}
}

func (r *RefundRepo) Create(
	ctx context.Context,
	refund *domain.RefundTransaction,
) error {
	_, err := r.col.InsertOne(ctx, refund)
	return err
}

func (r *RefundRepo) UpdateByMihPayID(
	ctx context.Context,
	mihPayID string,
	update bson.M,
) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"mihpayid": mihPayID},
		update,
	)
	return err
}
func (r *RefundRepo) FindByServiceID(
	ctx context.Context,
	serviceID string,
) (*domain.RefundTransaction, error) {

	var txn domain.RefundTransaction

	err := r.col.FindOne(
		ctx,
		bson.M{"serviceId": serviceID},
	).Decode(&txn)

	if err != nil {
		return nil, err
	}

	return &txn, nil
}

