package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type ResetTokenRepository interface {
	Save(ctx context.Context, token *domain.ResetToken) error
	Consume(ctx context.Context, tokenHash string) (*domain.ResetToken, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type resetTokenRepository struct {
	querier sqlc.Querier
}

func NewResetTokenRepository(querier sqlc.Querier) ResetTokenRepository {
	return &resetTokenRepository{
		querier: querier,
	}
}

func (r *resetTokenRepository) Save(ctx context.Context, token *domain.ResetToken) error {
	return r.querier.SavePasswordResetToken(ctx, sqlc.SavePasswordResetTokenParams{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	})
}

func (r *resetTokenRepository) Consume(ctx context.Context, tokenHash string) (*domain.ResetToken, error) {
	token, err := r.querier.ConsumePasswordResetToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
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

func (r *resetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if err := r.querier.DeletePasswordResetTokensByUserID(ctx, userID); err != nil {
		return err
	}

	return nil
}
