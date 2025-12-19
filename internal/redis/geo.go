package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func SaveProviderLocation(
	ctx context.Context,
	rdb *redis.Client,
	providerID string,
	lat, lon float64,
) error {
	return rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
		Name:      providerID,
		Latitude:  lat,
		Longitude: lon,
	}).Err()
}

func RemoveProvider(ctx context.Context, rdb *redis.Client, providerID string) error {
	return rdb.ZRem(ctx, "providers:geo", providerID).Err()
}
