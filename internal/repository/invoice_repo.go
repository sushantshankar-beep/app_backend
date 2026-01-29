package repository

import (
	"app_backend/internal/domain"
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type InvoiceRepo struct {
	col        *mongo.Collection
	serviceCol *mongo.Collection
}

func NewInvoiceRepo(db *mongo.Database) *InvoiceRepo {
	return &InvoiceRepo{
		col:        db.Collection("invoices"),
		serviceCol: db.Collection("accepted_services"),
	}
}

func (r *InvoiceRepo) Create(ctx context.Context, inv *domain.Invoice) error {
	if inv.ID.IsZero() {
		inv.ID = primitive.NewObjectID()
	}

	result, err := r.col.InsertOne(ctx, inv)
	if err != nil {
		return err
	}

	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		inv.ID = insertedID
	}

	return nil
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

func (r *InvoiceRepo) Update(ctx context.Context, id primitive.ObjectID, invoice *domain.Invoice) error {
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": invoice},
	)
	return err
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id string) (*domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.col.FindOne(ctx, bson.M{"serviceId": id}).Decode(&invoice)
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

func (r *InvoiceRepo) UpdatePDFUrl(ctx context.Context, id primitive.ObjectID, pdfUrl string) error {
	_, err := r.col.UpdateByID(
		ctx,
		id,
		bson.M{
			"$set": bson.M{
				"pdfUrl": pdfUrl,
			},
		},
	)
	log.Println("updatedddd urlll")
	return err
}
