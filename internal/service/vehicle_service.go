package service

import (
	"context"
	"encoding/json"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type MetaService struct {
	redis       *redis.Client
	brandRepo   *repository.VehicleBrandRepo
	serviceRepo *repository.ServiceMasterRepo
}

func NewMetaService(
	rdb *redis.Client,
	brandRepo *repository.VehicleBrandRepo,
	serviceRepo *repository.ServiceMasterRepo,
) *MetaService {
	return &MetaService{
		redis:       rdb,
		brandRepo:   brandRepo,
		serviceRepo: serviceRepo,
	}
}

/* ================= BRANDS ================= */

func (s *MetaService) GetBrands(
	ctx context.Context,
	vehicle string,
) ([]domain.VehicleBrand, error) {

	cacheKey := "meta:brands:" + vehicle

	if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var cached []domain.VehicleBrand
		if json.Unmarshal([]byte(val), &cached) == nil {
			return cached, nil
		}
	}

	brands, err := s.brandRepo.FindByVehicle(ctx, vehicle)
	if err != nil {
		return nil, err
	}

	go func() {
		b, _ := json.Marshal(brands)
		s.redis.Set(context.Background(), cacheKey, b, 24*time.Hour)
	}()

	return brands, nil
}

/* ================= SERVICES ================= */

func (s *MetaService) GetServices(
	ctx context.Context,
	vehicle string,
) ([]domain.ServiceMaster, error) {

	cacheKey := "meta:services:" + vehicle

	if val, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
		var cached []domain.ServiceMaster
		if json.Unmarshal([]byte(val), &cached) == nil {
			return cached, nil
		}
	}

	services, err := s.serviceRepo.FindByVehicle(ctx, vehicle)
	if err != nil {
		return nil, err
	}

	go func() {
		b, _ := json.Marshal(services)
		s.redis.Set(context.Background(), cacheKey, b, 24*time.Hour)
	}()

	return services, nil
}
