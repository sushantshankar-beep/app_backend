package service

import (
	"context"
	"errors"
	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KYCService struct {
	kycRepo      *repository.KYCRepo
	providerRepo *repository.ProviderRepo
}

func NewKYCService(kycRepo *repository.KYCRepo, providerRepo *repository.ProviderRepo) *KYCService {
	return &KYCService{
		kycRepo:      kycRepo,
		providerRepo: providerRepo,
	}
}

func (s *KYCService) CreateOrUpdateKYC(ctx context.Context, providerID string, req *domain.ProviderKYC) (*domain.ProviderKYC, error) {
	objID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, errors.New("invalid provider id")
	}

	existing, _ := s.kycRepo.FindByProviderID(ctx, providerID)

	req.ProviderID = objID
	req.Status = domain.KYC_PENDING

	if existing != nil {
		req.ID = existing.ID
		
		existingDocs := make(map[domain.DocumentType]domain.KYCDocument)
		for _, doc := range existing.Documents {
			existingDocs[doc.Type] = doc
		}

		for i := range req.Documents {
			if existingDoc, ok := existingDocs[req.Documents[i].Type]; ok {
				req.Documents[i].ID = existingDoc.ID
			} else if req.Documents[i].ID.IsZero() {
				req.Documents[i].ID = primitive.NewObjectID()
			}
			req.Documents[i].Verified = domain.VERIFICATION_PENDING
		}

		for docType, doc := range existingDocs {
			found := false
			for _, newDoc := range req.Documents {
				if newDoc.Type == docType {
					found = true
					break
				}
			}
			if !found {
				req.Documents = append(req.Documents, doc)
			}
		}
	} else {
		req.ID = primitive.NewObjectID()
		for i := range req.Documents {
			req.Documents[i].ID = primitive.NewObjectID()
			req.Documents[i].Verified = domain.VERIFICATION_PENDING
		}
	}

	err = s.kycRepo.Upsert(ctx, req)
	if err != nil {
		return nil, err
	}

	err = s.providerRepo.UpdateKYCID(ctx, providerID, req.ID)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (s *KYCService) GetKYC(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {
	return s.kycRepo.FindByProvider(ctx, providerID)
}