package api

import (
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
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
	log.Infof("Connected to Redis at %s", settings.RedisAddress)
	return ret, nil
}
