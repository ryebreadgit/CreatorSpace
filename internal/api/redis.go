package api

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

func initRedis(ctx context.Context) (*redis.Client, error) {
	ret := redis.NewClient(&redis.Options{
		Addr:     settings.RedisAddress,  // Change to your Redis server address and port
		Password: settings.RedisPassword, // Set the password if any
		DB:       settings.RedisDB,       // Use default DB
	})

	_, err := ret.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	fmt.Println("Connected to Redis successfully")
	return ret, nil
}
