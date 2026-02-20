package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/utils"

	"github.com/redis/go-redis/v9"
)

type ZoneService struct {
	zoneRepo *repository.ZoneRepository
	redis    *redis.Client
}

func NewZoneService(
	repo *repository.ZoneRepository,
	rdb *redis.Client,
) *ZoneService {
	return &ZoneService{
		zoneRepo: repo,
		redis:    rdb,
	}
}

func (s *ZoneService) CheckServiceable(
	ctx context.Context,
	lat float64,
	lon float64,
) (*domain.Zone, error) {

	cacheKey := fmt.Sprintf("zone:%f:%f", lat, lon)

	// =========================
	// 1️⃣ CHECK REDIS CACHE
	// =========================
	cachedCity, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cachedCity != "" {

		zone, err := s.zoneRepo.FindActiveByCity(ctx, cachedCity)
		if err != nil {
			return nil, err
		}

		return zone, nil
	}

	// =========================
	// 2️⃣ CALL OSM
	// =========================xq
	city, _, err := utils.ReverseGeocode(lat, lon)
	if err != nil {
		return nil, err
	}

	if city == "" {
		return nil, nil
	}

	// =========================
	// 3️⃣ NORMALIZE CITY
	// =========================
	city = normalizeCity(city)

	// =========================
	// 4️⃣ STORE IN REDIS (1 HOUR)
	// =========================
	_ = s.redis.Set(ctx, cacheKey, city, time.Hour).Err()

	// =========================
	// 5️⃣ CHECK DB (CASE INSENSITIVE)
	// =========================
	return s.zoneRepo.FindActiveByCity(ctx, city)
}

//
// 🔥 NORMALIZATION LOGIC
//
func normalizeCity(city string) string {

	city = strings.TrimSpace(strings.ToLower(city))

	return city
}
