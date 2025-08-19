package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)


var redDB *redis.Client

// Connection to redis
func ConnectRedis() {
	redisClient := redis.NewClient(
		&redis.Options{
			Addr: os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASS"),
			DB: 0,
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to redis: %v", err)
	}

	log.Println("Connected to the redis client successfully.")
	redDB = redisClient
}


// Returns a redis client
func GetRDB() *redis.Client {
	return redDB
}
