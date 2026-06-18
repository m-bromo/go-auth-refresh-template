package repository

import (
	"context"
	"fmt"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/redis/go-redis/v9"
)

type OtpRepository interface {
	SaveCode(ctx context.Context, email string, code string) error
	DeleteCode(ctx context.Context, email string) error
	GetCodeByEmail(ctx context.Context, email string) (string, error)
}

type otpRepository struct {
	redisClient *redis.Client
	cfg         *config.Config
}

func NewOtpRepository(redisClient *redis.Client, cfg *config.Config) OtpRepository {
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

func (r *otpRepository) DeleteCode(ctx context.Context, email string) error {
	_, err := r.redisClient.Del(ctx, email).Result()
	if err != nil {
		return fmt.Errorf("deleting otp code from redis: %w", err)
	}

	return nil
}

func (r *otpRepository) GetCodeByEmail(ctx context.Context, email string) (string, error) {
	code, err := r.redisClient.Get(ctx, email).Result()
	if err == redis.Nil {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("getting otp code from redis: %w", err)
	}

	return code, nil
}
