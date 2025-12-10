package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OTPRepo struct {
	col *mongo.Collection
}

func NewOTPRepo(db *mongo.Database) *OTPRepo {
	return &OTPRepo{col: db.Collection("otps")}
}
func (r *OTPRepo) Save(ctx context.Context, otp *domain.OTP) error {
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"phone": otp.Phone},
		bson.M{"$set": otp},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *OTPRepo) Find(ctx context.Context, phone, code string) (*domain.OTP, error) {
	var o domain.OTP
	err := r.col.FindOne(ctx, bson.M{"phone": phone, "code": code}).Decode(&o)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &o, err
}

func (r *OTPRepo) Delete(ctx context.Context, phone string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"phone": phone})
	return err
}

func boolPtr(b bool) *bool { return &b }
