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
	providerSvc  *ProviderService
}

func NewKYCService(kycRepo *repository.KYCRepo, providerRepo *repository.ProviderRepo, providerSvc *ProviderService) *KYCService {
	return &KYCService{
		kycRepo:      kycRepo,
		providerRepo: providerRepo,
		providerSvc:  providerSvc,
	}
}

func (s *KYCService) CreateOrUpdateKYC(
	ctx context.Context,
	providerID string,
	req *domain.ProviderKYC,
) (*domain.ProviderKYC, error) {

	objID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return nil, errors.New("invalid provider id")
	}

	existing, _ := s.kycRepo.FindByProviderID(ctx, providerID)

	req.ProviderID = objID
	
	if existing == nil {
		req.Status = domain.KYC_PENDING
	} else {
		req.ID = existing.ID
		req.Status = existing.Status 
	}

	if existing != nil {
		existingDocs := make(map[domain.DocumentType]domain.KYCDocument)
		for _, doc := range existing.Documents {
			existingDocs[doc.Type] = doc
		}

		newDocsUploaded := false
		for i := range req.Documents {
			if existingDoc, ok := existingDocs[req.Documents[i].Type]; ok {
				req.Documents[i].ID = existingDoc.ID
			} else if req.Documents[i].ID.IsZero() {
				req.Documents[i].ID = primitive.NewObjectID()
			}

			req.Documents[i].Verified = domain.VERIFICATION_PENDING
			newDocsUploaded = true
		}

		if newDocsUploaded {
			req.Status = domain.KYC_PENDING
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

	if err := s.kycRepo.Upsert(ctx, req); err != nil {
		return nil, err
	}

	if err := s.providerRepo.UpdateKYCID(ctx, providerID, req.ID); err != nil {
		return nil, err
	}

	if provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(providerID)); err == nil && provider != nil {
		_ = s.providerSvc.generateAndUploadAgreement(ctx, provider)
		_ = s.providerRepo.UpdateAgreement(ctx, provider)
	}
   
	return req, nil
}


func (s *KYCService) GetKYC(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {
	return s.kycRepo.FindByProvider(ctx, providerID)
}