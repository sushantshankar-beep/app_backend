package repository

import (
	"context"
	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/mongo"
)

type CancellationRepo struct {
	col *mongo.Collection
}

func NewCancellationRepo(db *mongo.Database) *CancellationRepo {
	return &CancellationRepo{
		col: db.Collection("cancellation_logs"),
	}
}

func (r *CancellationRepo) Insert(ctx context.Context, log *domain.CancellationLog) {
	_, _ = r.col.InsertOne(ctx, log)
}
