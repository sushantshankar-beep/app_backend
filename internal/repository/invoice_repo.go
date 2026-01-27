package repository

import (
	"app_backend/internal/domain"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type InvoiceRepo struct {
	col *mongo.Collection
	serviceCol  *mongo.Collection
}

func NewInvoiceRepo(db *mongo.Database) *InvoiceRepo {
	return &InvoiceRepo{
		col:  db.Collection("invoices"),
		serviceCol:  db.Collection("accepted_services"),
	}
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

func (r *InvoiceRepo) GetOrCreateInvoice(ctx context.Context, serviceID primitive.ObjectID) (*domain.Invoice, error) {
	var invoice domain.Invoice
	log.Println("skjcqebjqbjc",serviceID)
	err := r.col.FindOne(ctx, bson.M{"serviceId": serviceID}).Decode(&invoice)
	if err == nil {
		return &invoice, nil
	}
	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	return r.createInvoiceFromService(ctx, serviceID)
}



func (r *InvoiceRepo) createInvoiceFromService(ctx context.Context, serviceID primitive.ObjectID) (*domain.Invoice, error) {
	log.Println("jsxdbc jsbd",serviceID)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": serviceID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "user",
			"foreignField": "_id",
			"as":           "userInfo",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "providers",
			"localField":   "provider",
			"foreignField": "_id",
			"as":           "providerInfo",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "service_requests",
			"localField":   "serviceRequest",
			"foreignField": "_id",
			"as":           "requestInfo",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path": "$userInfo",
			"preserveNullAndEmptyArrays": true,
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path": "$providerInfo",
			"preserveNullAndEmptyArrays": true,
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path": "$requestInfo",
			"preserveNullAndEmptyArrays": true,
		}}},
		
	}

	cursor, err := r.serviceCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result struct {
		ID           primitive.ObjectID `bson:"_id"`
		User         primitive.ObjectID `bson:"user"`
		Provider     primitive.ObjectID `bson:"provider"`
		FinalPrice   float64            `bson:"finalPrice"`
		CompletedAt  *time.Time         `bson:"completedAt"`
		UserInfo     UserData           `bson:"userInfo"`
		ProviderInfo ProviderData       `bson:"providerInfo"`
		RequestInfo  RequestData        `bson:"requestInfo"`
	}

	if !cursor.Next(ctx) {
		return nil, mongo.ErrNoDocuments
	}

	if err := cursor.Decode(&result); err != nil {
		return nil, err
	}

	invoiceNumber := fmt.Sprintf("INV-%s", strings.ToUpper(result.ID.Hex()[len(result.ID.Hex())-8:]))

	gstPercent := 18.0
	serviceCharge := result.FinalPrice / (1 + gstPercent/100)
	taxAmount := result.FinalPrice - serviceCharge

	invoice := &domain.Invoice{
		ID:              primitive.NewObjectID(),
		InvoiceNumber:   invoiceNumber,
		UserID:          result.User,
		ServiceID:       result.ID,
		ProviderID:      result.Provider,
		BillToName:      result.UserInfo.Name,
		BillToAddress:   result.UserInfo.Address,
		BillToPhone:     result.UserInfo.Phone,
		ProviderName:    getProviderName(result.ProviderInfo),
		ProviderAddress: result.ProviderInfo.Address,
		ProviderPhone:   result.ProviderInfo.Phone,
		ProviderGST:     result.ProviderInfo.GSTNumber,
		VehicleBrand:    result.RequestInfo.Brand,
		VehicleModel:    result.RequestInfo.Model,
		VehicleNumber:   result.RequestInfo.VehicleNumber,
		VehicleYear:     result.RequestInfo.Year,
		VehicleType:     result.RequestInfo.VehicleType,
		FuelType:        result.RequestInfo.FuelType,
		ServiceType:     result.RequestInfo.ServiceType,
		ServiceDate:     result.CompletedAt,
		Items: []domain.InvoiceItem{
			{
				Name:        result.RequestInfo.ServiceType,
				Qty:         1,
				Price:       serviceCharge,
				GSTPercent:  gstPercent,
				GrossAmount: result.FinalPrice,
			},
		},
		SubTotal:    serviceCharge,
		TaxAmount:   taxAmount,
		TotalAmount: result.FinalPrice,
		CreatedAt:   time.Now(),
	}

	_, err = r.col.InsertOne(ctx, invoice)
	if err != nil {
		return nil, err
	}

	return invoice, nil
}

func getProviderName(provider ProviderData) string {
	if provider.CompanyName != "" {
		return provider.CompanyName
	}
	return provider.Name
}

type UserData struct {
	Name    string `bson:"name"`
	Phone   string `bson:"phone"`
	Address string `bson:"address"`
}

type ProviderData struct {
	Name        string `bson:"name"`
	CompanyName string `bson:"companyName"`
	Phone       string `bson:"phone"`
	Address     string `bson:"address"`
	GSTNumber   string `bson:"gstNumber"`
}

type RequestData struct {
	Brand         string `bson:"brand"`
	Model         string `bson:"model"`
	Year          int    `bson:"year"`
	VehicleType   string `bson:"vehicleType"`
	VehicleNumber string `bson:"vehicleNumber"`
	FuelType      string `bson:"fuelType"`
	ServiceType   string `bson:"serviceType"`
}