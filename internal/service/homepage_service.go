package service

import (
	"context"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/ports"
	"app_backend/internal/validation"
)

type HomepageService struct {
	repo ports.HomepageRepository
}

func NewHomepageService(repo ports.HomepageRepository) *HomepageService {
	return &HomepageService{repo: repo}
}

func (s *HomepageService) GetHomepage(ctx context.Context, id string) (*domain.Homepage, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *HomepageService) CreateOrUpdateHomepage(ctx context.Context, req validation.HomepageRequest) (*domain.Homepage, error) {
	homepage := &domain.Homepage{}
	isUpdate := false

	if req.ID != "" {
		existing, err := s.repo.FindByID(ctx, req.ID)
		if err != nil {
			if err == domain.ErrNotFound {
				return nil, domain.ErrNotFound
			}
			return nil, err
		}
		homepage = existing
		isUpdate = true
	}

	homepage.UserCohort = req.UserCohort
	homepage.Location = domain.LocationInfo{
		City:             req.Location.City,
		Address:          req.Location.Address,
		Latitude:         req.Location.Latitude,
		Longitude:        req.Location.Longitude,
		Pincode:          req.Location.Pincode,
		FormattedAddress: req.Location.FormattedAddress,
	}

	homepage.Banners = make([]domain.Banner, len(req.Banners))
	for i, b := range req.Banners {
		homepage.Banners[i] = domain.Banner{
			Title:             b.Title,
			ImageURL:          b.ImageURL,
			RedirectionURL:    b.RedirectionURL,
			RedirectionParams: b.RedirectionParams,
		}
	}

	homepage.Categories = make([]domain.Category, len(req.Categories))
	for i, c := range req.Categories {
		homepage.Categories[i] = domain.Category{
			Name:           c.Name,
			IconURL:        c.IconURL,
			RedirectionURL: c.RedirectionURL,
		}
	}

	homepage.IsActive = req.IsActive

	now := time.Now()
	if !isUpdate {
		homepage.CreatedAt = now
		homepage.UpdatedAt = now
		return homepage, s.repo.Create(ctx, homepage)
	}

	homepage.UpdatedAt = now
	return homepage, s.repo.Update(ctx, homepage)
}