package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type FindMechanicService struct {
	rdb    *redis.Client
	socket SocketEmitter
}

type SocketEmitter interface {
	EmitWithRetry(room, event string, payload any, retry int)
}

func NewFindMechanicService(rdb *redis.Client, socket SocketEmitter) *FindMechanicService {
	return &FindMechanicService{rdb: rdb, socket: socket}
}

func (s *FindMechanicService) Find(
	ctx context.Context,
	requestID string,
	lat, lng float64,
) error {

	// 🔒 Deduplicate user taps
	lockKey := "request:" + requestID + ":lock"
	ok, _ := s.rdb.SetNX(ctx, lockKey, 1, 30*time.Second).Result()
	if !ok {
		return nil
	}

	radius := []float64{10, 20}

	for _, km := range radius {
		providers, _ := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			lng, lat,
			&redis.GeoRadiusQuery{
				Radius:    km,
				Unit:      "km",
				WithCoord: true,
			},
		).Result()

		if len(providers) == 0 {
			continue
		}

		for _, p := range providers {
			room := "provider:" + p.Name
			s.socket.EmitWithRetry(
				room,
				"bid:request",
				map[string]any{
					"requestId": requestID,
					"distance":  km,
				},
				2,
			)
		}
		return nil
	}

	return nil
}
