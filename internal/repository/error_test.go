package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

type querierStub struct {
	sqlc.Querier
	getUserByIDFunc             func(context.Context, uuid.UUID) (sqlc.User, error)
	consumePasswordResetFunc    func(context.Context, string) (sqlc.PasswordResetToken, error)
	getRefreshTokenByIDFunc     func(context.Context, uuid.UUID) (sqlc.RefreshToken, error)
	consumeRefreshTokenByIDFunc func(context.Context, uuid.UUID) (sqlc.RefreshToken, error)
}

func (q *querierStub) GetUserByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return q.getUserByIDFunc(ctx, id)
}

func (q *querierStub) ConsumePasswordResetToken(
	ctx context.Context,
	tokenHash string,
) (sqlc.PasswordResetToken, error) {
	return q.consumePasswordResetFunc(ctx, tokenHash)
}

func (q *querierStub) GetRefreshTokenByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.RefreshToken, error) {
	return q.getRefreshTokenByIDFunc(ctx, id)
}

func (q *querierStub) ConsumeRefreshToken(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.RefreshToken, error) {
	return q.consumeRefreshTokenByIDFunc(ctx, id)
}

func TestRepositoriesWrapDriverErrors(t *testing.T) {
	driverErr := errors.New("driver failed")

	tests := []struct {
		name        string
		operation   func() error
		wantContext string
	}{
		{
			name: "user repository",
			operation: func() error {
				querier := &querierStub{
					getUserByIDFunc: func(context.Context, uuid.UUID) (sqlc.User, error) {
						return sqlc.User{}, driverErr
					},
				}
				_, err := repository.NewSqlcUserRepository(querier).GetByID(t.Context(), uuid.New())
				return err
			},
			wantContext: "getting user by ID",
		},
		{
			name: "reset token repository",
			operation: func() error {
				querier := &querierStub{
					consumePasswordResetFunc: func(
						context.Context,
						string,
					) (sqlc.PasswordResetToken, error) {
						return sqlc.PasswordResetToken{}, driverErr
					},
				}
				_, err := repository.NewSqlcResetTokenRepository(querier).Consume(t.Context(), "hash")
				return err
			},
			wantContext: "consuming password reset token",
		},
		{
			name: "refresh token repository",
			operation: func() error {
				querier := &querierStub{
					getRefreshTokenByIDFunc: func(
						context.Context,
						uuid.UUID,
					) (sqlc.RefreshToken, error) {
						return sqlc.RefreshToken{}, driverErr
					},
				}
				_, err := repository.NewSqlcRefreshTokenRepository(querier).Get(t.Context(), uuid.New())
				return err
			},
			wantContext: "getting refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.operation()

			if !errors.Is(err, driverErr) {
				t.Fatalf("error = %v, want wrapped driver error", err)
			}
			if !strings.Contains(err.Error(), tt.wantContext) {
				t.Errorf("error = %q, want context %q", err, tt.wantContext)
			}
		})
	}
}

func TestRefreshTokenRepositoryRecognizesWrappedNoRows(t *testing.T) {
	noRowsErr := fmt.Errorf("querying driver: %w", sql.ErrNoRows)
	querier := &querierStub{
		getRefreshTokenByIDFunc: func(context.Context, uuid.UUID) (sqlc.RefreshToken, error) {
			return sqlc.RefreshToken{}, noRowsErr
		},
		consumeRefreshTokenByIDFunc: func(context.Context, uuid.UUID) (sqlc.RefreshToken, error) {
			return sqlc.RefreshToken{}, noRowsErr
		},
	}
	refreshTokens := repository.NewSqlcRefreshTokenRepository(querier)

	token, err := refreshTokens.Get(t.Context(), uuid.New())
	if err != nil || token != nil {
		t.Errorf("Get() = (%v, %v), want (nil, nil)", token, err)
	}

	userID, err := refreshTokens.Consume(t.Context(), uuid.New())
	if err != nil || userID != "" {
		t.Errorf("Consume() = (%q, %v), want (empty, nil)", userID, err)
	}
}
