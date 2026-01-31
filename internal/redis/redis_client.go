package redis

import (
	"context"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	rdb  *redis.Client
	once sync.Once
)

func NewRedis() *redis.Client {

	once.Do(func() {

		rdb = redis.NewClient(&redis.Options{
			Addr:     "redis-14559.c14.us-east-1-2.ec2.cloud.redislabs.com:14559",
			Username: "default",
			Password: "xDPOi04pp75lgZYoOowNuP0IAQmwUpx1",
			DB:       0,

			// 🔥 VERY IMPORTANT
			PoolSize:     50,
			MinIdleConns: 10,
		})

		ctx := context.Background()

		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Fatal("❌ Redis connection failed:", err)
		}

		log.Println("✅ Redis connected")
	})

	return rdb
}
