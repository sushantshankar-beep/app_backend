package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/s3"
	// "app_backend/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KYCService struct {
	repo     *repository.KYCRepo
	uploader *s3.Uploader
}

func NewKYCService(repo *repository.KYCRepo) *KYCService {
	uploader, _ := s3.NewUploader()
	return &KYCService{repo: repo, uploader: uploader}
}

func (s *KYCService) SubmitKYC(ctx context.Context, providerID string, req *domain.ProviderKYC) error {
	objID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return errors.New("invalid provider id")
	}

	req.ProviderID = objID
	req.Status = domain.KYC_PENDING
	req.UpdatedAt = time.Now()

	return s.repo.Upsert(ctx, req)
}

func (s *KYCService) GetKYC(ctx context.Context, providerID string) (*domain.ProviderKYC, error) {
	return s.repo.FindByProvider(ctx, providerID)
}
