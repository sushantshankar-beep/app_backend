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
			Addr:     "redis-17553.c274.us-east-1-3.ec2.cloud.redislabs.com:17553",
			Username: "default",
			Password: "1DubYPzF4ADKzESuG1QUQczPVYs1SnIc",
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
