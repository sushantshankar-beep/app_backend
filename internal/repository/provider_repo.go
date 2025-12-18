package repository

import (
	"context"

	"app_backend/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProviderRepo struct {
	col *mongo.Collection
}

func (r *ProviderRepo) GetFCMToken(ctx context.Context,id primitive.ObjectID,) (string, error) {

		var result struct {
			FCMToken string `bson:"fcmToken"`
		}

		err := r.col.FindOne(
			ctx,
			bson.M{"_id": id},
		).Decode(&result)

		return result.FCMToken, err
	}

func NewProviderRepo(db *mongo.Database) *ProviderRepo {
	return &ProviderRepo{col: db.Collection("providerschemas")}
}

func (r *ProviderRepo) FindByPhone(ctx context.Context, phone string) (*domain.Provider, error) {
	var p domain.Provider
	err := r.col.FindOne(ctx, bson.M{"phone": phone}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, domain.ErrNotFound
	}
	return &p, err
}

func (r *ProviderRepo) AddComplaint(ctx context.Context, providerID primitive.ObjectID, complaintID primitive.ObjectID) error {
	_, err := r.col.UpdateByID(ctx, providerID, bson.M{
		"$push": bson.M{"complaints": complaintID},
	})
	return err
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

func (r *ProviderRepo) Create(ctx context.Context, p *domain.Provider) error {
	res, err := r.col.InsertOne(ctx, p)
	if err != nil {
		return err
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		p.ID = domain.ProviderID(oid.Hex())
	}
	return nil
}

func (r *ProviderRepo) Update(ctx context.Context, p *domain.Provider) error {
	objID, err := primitive.ObjectIDFromHex(string(p.ID))
	if err != nil {
		return err
	}
	
	update := bson.M{
		"$set": bson.M{
			"name":               p.Name,
			"email":              p.Email,
			"alternateContact":   p.AlternateContact,
			"profileUrl":         p.ProfileURL,
			"address":            p.Address,
			"permanentAddress":   p.PermanentAddress,
			"city":               p.City,
			"GSTNumber":          p.GSTNumber,
			"identityProof":      p.IdentityProof,
			"addressProof":       p.AddressProof,
			"cancelCheque":       p.CancelCheque,
			"bankDetails":        p.BankDetails,
			"vehicleNumber":      p.VehicleNumber,
			"formSubmitted":      p.FormSubmitted,
			"description":        p.Description,
			"vehicleType":        p.VehicleType,
			"providerBrands":     p.ProviderBrands,
			"providerServices":   p.ProviderServices,
			"companyName":        p.CompanyName,
			"updatedAt":          p.UpdatedAt,
		},
	}
	
	_, err = r.col.UpdateByID(ctx, objID, update)
	return err
}
func (r *ProviderRepo) FindOne(
	ctx context.Context,
	filter bson.M,
	result any,
) error {
	return r.col.FindOne(ctx, filter).Decode(result)
}
