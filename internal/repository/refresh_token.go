package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	Get(ctx context.Context, tokenID uuid.UUID) (string, error)
	Delete(ctx context.Context, tokenID uuid.UUID) error
}

type refreshTokenRepository struct {
	redisClient *redis.Client
	cfg         *config.Config
}

func NewRefreshTokenRepository(redisClient *redis.Client, cfg *config.Config) RefreshTokenRepository {
	return &refreshTokenRepository{
		redisClient: redisClient,
		cfg:         cfg,
	}
}

func (r *refreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.redisClient.Set(ctx, token.ID.String(), token.UserID.String(), r.cfg.RefreshToken.Duration).Result()
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *refreshTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (string, error) {
	userID, err := r.redisClient.Get(ctx, tokenID.String()).Result()
	if err == redis.Nil {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("get refresh token: %w", err)
	}

	return userID, nil
}

func (r *refreshTokenRepository) Delete(ctx context.Context, tokenID uuid.UUID) error {
	if _, err := r.redisClient.Del(ctx, tokenID.String()).Result(); err != nil {
		return fmt.Errorf("delete token: %w", err)
	}

	return nil
}
