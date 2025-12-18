package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedis() *redis.Client {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis-10113.c16.us-east-1-2.ec2.cloud.redislabs.com:10113",
		Username: "default",
		Password: "jDk9eSng4XRj2yqE81Pp1oml44LEIsZi",
		DB:       0,
	})

	rdb.Set(ctx, "foo", "bar", 0)
	result, err := rdb.Get(ctx, "foo").Result()

	if err != nil {
		panic(err)
	}

	fmt.Println(result) // >>> bar
	return rdb

}