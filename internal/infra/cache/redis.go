package cache

import (
	"context"
	"fmt"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(redisOptions *configs.Redis) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisOptions.Host, redisOptions.Port),
		Password: redisOptions.Password,
		DB:       0,
		Protocol: 2,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return redisClient, nil
}
