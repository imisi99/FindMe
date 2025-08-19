package database

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)


var RedDB *redis.Client

// Connection to redis
func ConnectRedis() {
	redisClient := redis.NewClient(
		&redis.Options{
			Addr: os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_KEY"),
			DB: 0,
		},
	)

	ctx := context.Background()

	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to redis: %v", err)
	}
	log.Println("Connected to the redis client successfully.")
	RedDB = redisClient
}


// Returns a redis client
func GetRDB() *redis.Client {
	return RedDB
}
