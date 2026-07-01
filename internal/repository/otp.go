package repository

import (
	"context"
	"fmt"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/redis/go-redis/v9"
)

type OtpRepository interface {
	SaveCode(ctx context.Context, email string, code string) error
	ConsumeCodeIfMatches(ctx context.Context, email string, code string) (bool, error)
}

type otpRepository struct {
	redisClient *redis.Client
	cfg         *configs.Config
}

const consumeCodeMaxRetries = 3

func NewOtpRepository(redisClient *redis.Client, cfg *configs.Config) OtpRepository {
	return &otpRepository{
		redisClient: redisClient,
		cfg:         cfg,
	}
}

func (r *otpRepository) SaveCode(ctx context.Context, email string, code string) error {
	_, err := r.redisClient.Set(ctx, email, code, r.cfg.OTP.Duration).Result()
	if err != nil {
		return fmt.Errorf("saving otp code to redis: %w", err)
	}

	return nil
}

func (r *otpRepository) ConsumeCodeIfMatches(ctx context.Context, email string, code string) (bool, error) {
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
