package repository

import (
	"context"
	"fmt"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/redis/go-redis/v9"
)

type RedisOtpRepository struct {
	redisClient *redis.Client
	otpOptions  *configs.OTP
}

const consumeCodeMaxRetries = 3

func NewRedisOtpRepository(redisClient *redis.Client, otpOptions *configs.OTP) *RedisOtpRepository {
	return &RedisOtpRepository{
		redisClient: redisClient,
		otpOptions:  otpOptions,
	}
}

func (r *RedisOtpRepository) SaveCode(ctx context.Context, email string, code string) error {
	_, err := r.redisClient.Set(ctx, email, code, r.otpOptions.Duration).Result()
	if err != nil {
		return fmt.Errorf("saving otp code to redis: %w", err)
	}

	return nil
}

func (r *RedisOtpRepository) ConsumeCodeIfMatches(ctx context.Context, email string, code string) (bool, error) {
	var consumed bool

	for range consumeCodeMaxRetries {
		err := r.redisClient.Watch(ctx, func(tx *redis.Tx) error {
			storedCode, err := tx.Get(ctx, email).Result()
			if err == redis.Nil {
				consumed = false
				return nil
			}

			if err != nil {
				return err
			}

			if storedCode != code {
				consumed = false
				return nil
			}

			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, email)
				return nil
			})
			if err != nil {
				return err
			}

			consumed = true
			return nil
		}, email)

		if err == redis.TxFailedErr {
			continue
		}

		if err != nil {
			return false, fmt.Errorf("consuming otp code from redis: %w", err)
		}

		return consumed, nil
	}

	return false, fmt.Errorf("consuming otp code from redis: %w", redis.TxFailedErr)
}
