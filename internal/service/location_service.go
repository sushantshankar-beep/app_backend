package service

import (
	"context"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
)

type LocationService struct {
	repo *repository.LocationRepo
}

func NewLocationService(repo *repository.LocationRepo) *LocationService {
	return &LocationService{repo: repo}
}

func (s *LocationService) SaveLocation(ctx context.Context, loc *domain.Location) error {
	return s.repo.Save(ctx, loc)
}

func (s *LocationService) GetLocation(ctx context.Context, userID string) (*domain.Location, error) {
	return s.repo.GetByUser(ctx, userID)
}
