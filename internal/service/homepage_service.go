package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"github.com/redis/go-redis/v9"
)

type HomepageService struct {
	repo  *repository.HomepageRepo
	redis *redis.Client
}

func NewHomepageService(
	repo *repository.HomepageRepo,
	redis *redis.Client,
) *HomepageService {
	return &HomepageService{
		repo:  repo,
		redis: redis,
	}
}

func (s *HomepageService) GetHomepage(
	ctx context.Context,
	country, state, city string,
) (*domain.Homepage, error) {

	cacheKey := fmt.Sprintf("homepage:%s:%s:%s", country, state, city)
	if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var cached domain.Homepage
		if json.Unmarshal([]byte(val), &cached) == nil {
			return &cached, nil
		}
	}

	homepage, err := s.repo.FindByLocation(ctx, country, state, city)
	if err != nil {
		return nil, err
	}

	// async cache
	go func() {
		b, _ := json.Marshal(homepage)
		s.redis.Set(context.Background(), cacheKey, b, 10*time.Minute)
	}()

	return homepage, nil
}
