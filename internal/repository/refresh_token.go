package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type SqlcRefreshTokenRepository struct {
	querier sqlc.Querier
}

func NewSqlcRefreshTokenRepository(querier sqlc.Querier) *SqlcRefreshTokenRepository {
	return &SqlcRefreshTokenRepository{
		querier: querier,
	}
}

func (r *SqlcRefreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	if err := r.querier.SaveRefreshToken(ctx, sqlc.SaveRefreshTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}

	return nil
}

func (r *SqlcRefreshTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error) {
	token, err := r.querier.GetRefreshTokenByID(ctx, tokenID)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	return &domain.RefreshToken{
		ID:        token.ID,
		UserID:    token.UserID,
		CreatedAt: token.CreatedAt,
		ExpiresAt: token.ExpiresAt,
	}, nil
}

func (r *SqlcRefreshTokenRepository) Consume(ctx context.Context, tokenID uuid.UUID) (string, error) {
	token, err := r.querier.ConsumeRefreshToken(ctx, tokenID)
	if err == sql.ErrNoRows {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("consume refresh token: %w", err)
	}

	return token.UserID.String(), nil
}

func (r *SqlcRefreshTokenRepository) Delete(ctx context.Context, tokenID uuid.UUID) error {
	if err := r.querier.DeleteRefreshToken(ctx, tokenID); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	return nil
}

func (r *SqlcRefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.querier.DeleteRefreshTokensByUserID(ctx, userID); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}

	return nil
}
