package repository

import (
	"context"

	"app_backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type InvoiceRepo struct {
	col *mongo.Collection
}

func NewInvoiceRepo(db *mongo.Database) *InvoiceRepo {
	return &InvoiceRepo{col: db.Collection("invoices")}
}

func (r *InvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	_, err := r.col.InsertOne(ctx, inv)
	return err
}

func (r *InvoiceRepo) FindByServiceID(
	ctx context.Context,
	serviceID string,
) (*domain.Invoice, error) {

	var inv domain.Invoice
	err := r.col.FindOne(ctx, bson.M{"serviceId": serviceID}).Decode(&inv)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
func (r *InvoiceRepo) FindByNumber(ctx context.Context, no string) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.col.FindOne(ctx, bson.M{"invoiceNumber": no}).Decode(&inv)
	return &inv, err
}