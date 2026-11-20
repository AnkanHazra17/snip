package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func newRedisClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:          os.Getenv("REDIS_ADDR"),
		DialTimeout:   200 * time.Millisecond,
		ReadTimeout:   100 * time.Millisecond,
		WriteTimeout:  100 * time.Millisecond,
		MaxRetries:    -1,
		DialerRetries: 1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("WARNING: failed to connect to Redis: %v", err)
	} else {
		log.Println("connected to redis")
	}

	return rdb
}
