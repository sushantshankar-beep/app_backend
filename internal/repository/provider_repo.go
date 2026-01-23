package repository

import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Provider Repository
Collection: providerschemas
*/
type ProviderRepo struct {
	col *mongo.Collection
}

/* ---------------- CONSTRUCTOR ---------------- */

func NewProviderRepo(db *mongo.Database) *ProviderRepo {
	return &ProviderRepo{
		col: db.Collection("providerschemas"),
	}
}

/* ---------------- HELPERS ---------------- */

func generateProviderCode(seq int64) string {
	return fmt.Sprintf("PRV%05d", seq)
}

/* ---------------- QUERIES ---------------- */

func (r *ProviderRepo) FindByPhone(ctx context.Context, phone string) (*domain.Provider, error) {
	var p domain.Provider
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *ProviderRepo) FindByID(ctx context.Context, id domain.ProviderID) (*domain.Provider, error) {
	objID, err := primitive.ObjectIDFromHex(string(id))
	if err != nil {
		return nil, domain.ErrNotFound
	}

	var provider domain.Provider
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&provider)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	provider.ID = domain.ProviderID(objID.Hex())
	return &provider, nil
}

/* ---------------- CREATE ---------------- */

func (r *ProviderRepo) Create(ctx context.Context, p *domain.Provider) error {

	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	p.IsActive = "inactive"
	p.Rating = "0"

	res, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return err
	}

	p.ID = domain.ProviderID(res.InsertedID.(primitive.ObjectID).Hex())
	return nil
}

/* ---------------- UPDATE PROFILE ---------------- */

func (r *ProviderRepo) Update(ctx context.Context, p *domain.Provider) error {

	objID, err := primitive.ObjectIDFromHex(string(p.ID))
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"name":             p.Name,
			"companyName":      p.CompanyName,
			"email":            p.Email,
			"alternateContact": p.AlternateContact,
			"profileUrl":       p.ProfileURL,
			"address":          p.Address,
			"permanentAddress": p.PermanentAddress,
			"city":             p.City,
			"vehicleNumber": p.VehicleNumber,
			"description":   p.Description,
			"vehicleType":      p.VehicleType,
			"providerBrands":   p.ProviderBrands,
			"providerServices": p.ProviderServices,
			"formSubmitted": p.FormSubmitted,
			"isActive":      p.IsActive,
			"updatedAt": time.Now(),
		},
	}

	_, err = r.col.UpdateByID(ctx, objID, update)
	return err
}

/* ---------------- FCM TOKEN ---------------- */

func (r *ProviderRepo) GetFCMToken(
	ctx context.Context,
	id primitive.ObjectID,
) (string, error) {

	var result struct {
		FCMToken string `bson:"fcmToken"`
	}

	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	return result.FCMToken, err
}

/* ---------------- COMPLAINTS ---------------- */

func (r *ProviderRepo) AddComplaint(
	ctx context.Context,
	providerID primitive.ObjectID,
	complaintID primitive.ObjectID,
) error {

	_, err := r.col.UpdateByID(
		ctx,
		providerID,
		bson.M{"$push": bson.M{"complaints": complaintID}},
	)
	return err
}

/* ---------------- GENERIC FIND ---------------- */

func (r *ProviderRepo) FindOne(
	ctx context.Context,
	filter bson.M,
	result any,
) error {
	return r.col.FindOne(ctx, filter).Decode(result)
}

func (r *ProviderRepo) UpdateKYCID(ctx context.Context, providerID string, kycID primitive.ObjectID) error {

	objID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return err
	}
	
	filter := bson.M{"_id": objID}
	update := bson.M{
		"$set": bson.M{
			"kycId":     kycID,
			"updatedAt": time.Now(),
		},
	}
	
	_, err = r.col.UpdateOne(ctx, filter, update)
	return err
}