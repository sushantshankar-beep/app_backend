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
			Addr:     "redis-17094.c62.us-east-1-4.ec2.cloud.redislabs.com:17094",
			Username: "default",
			Password: "KZkWrxXIM3GZwEDTSOOIVNGXAn1m8qsS",
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
