package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type SqlcResetTokenRepository struct {
	querier sqlc.Querier
}

func NewSqlcResetTokenRepository(querier sqlc.Querier) *SqlcResetTokenRepository {
	return &SqlcResetTokenRepository{
		querier: querier,
	}
}

func (r *SqlcResetTokenRepository) Save(ctx context.Context, token *domain.ResetToken) error {
	if err := r.querier.SavePasswordResetToken(ctx, sqlc.SavePasswordResetTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("saving password reset token: %w", err)
	}

	return nil
}

func (r *SqlcResetTokenRepository) Consume(ctx context.Context, tokenHash string) (*domain.ResetToken, error) {
	token, err := r.querier.ConsumePasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("consuming password reset token: %w", err)
	}

	return &domain.ResetToken{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		UsedAt:    token.UsedAt.Time,
		CreatedAt: token.CreatedAt,
	}, nil
}

func (r *SqlcResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.querier.DeletePasswordResetTokensByUserID(ctx, userID); err != nil {
		return fmt.Errorf("deleting password reset tokens by user ID: %w", err)
	}

	return nil
}
