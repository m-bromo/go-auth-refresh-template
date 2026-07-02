package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type SqlcOtpRepository struct {
	querier    sqlc.Querier
	otpOptions *configs.OTP
}

func NewSqlcOtpRepository(
	querier sqlc.Querier,
	otpOptions *configs.OTP,
) *SqlcOtpRepository {
	return &SqlcOtpRepository{
		querier:    querier,
		otpOptions: otpOptions,
	}
}

func (r *SqlcOtpRepository) Save(ctx context.Context, otp *domain.OTP) error {
	if err := r.querier.SaveOtpCode(ctx, sqlc.SaveOtpCodeParams{
		ID:         otp.ID,
		Identifier: otp.Identifier,
		CodeHash:   otp.Code,
		Attempts:   int16(otp.Attempts),
		ExpiresAt:  otp.ExpiresAt,
		CreatedAt:  otp.CreatedAt,
	}); err != nil {
		return fmt.Errorf("saving otp code: %w", err)
	}

	return nil
}

func (r *SqlcOtpRepository) InvalidateByIdentifier(ctx context.Context, identifier string) error {
	if err := r.querier.InvalidateOtpCodesByIdentifier(ctx, identifier); err != nil {
		return fmt.Errorf("invalidating otp codes by identifier: %w", err)
	}

	return nil
}

func (r *SqlcOtpRepository) Consume(
	ctx context.Context,
	email string,
	codeHash string,
) (*domain.OTP, error) {
	otp, err := r.querier.ConsumeOtpCode(ctx, sqlc.ConsumeOtpCodeParams{
		CodeHash:   codeHash,
		Identifier: email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("fetching otp code: %w", err)
	}

	return &domain.OTP{
		ID:         otp.ID,
		Identifier: otp.Identifier,
		Code:       otp.CodeHash,
		ExpiresAt:  otp.ExpiresAt,
		CreatedAt:  otp.CreatedAt,
	}, nil
}

func (r *SqlcOtpRepository) ConsumeByChallengeID(
	ctx context.Context,
	challengeID uuid.UUID,
	codeHash string,
) (*domain.OTP, error) {
	otp, err := r.querier.ConsumeOtpCodeByChallengeID(ctx, sqlc.ConsumeOtpCodeByChallengeIDParams{
		ID:          challengeID,
		CodeHash:    codeHash,
		MaxAttempts: int16(r.otpOptions.MaxAttempts),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("fetching otp code by challenge id: %w", err)
	}

	return &domain.OTP{
		ID:         otp.ID,
		Identifier: otp.Identifier,
		Code:       otp.CodeHash,
		Attempts:   int(otp.Attempts),
		ExpiresAt:  otp.ExpiresAt,
		CreatedAt:  otp.CreatedAt,
	}, nil
}

func (r *SqlcOtpRepository) IncreaseAttempts(ctx context.Context, challengeID uuid.UUID) error {
	if err := r.querier.IncreaseOtpAttempts(ctx, sqlc.IncreaseOtpAttemptsParams{
		ID:          challengeID,
		MaxAttempts: int16(r.otpOptions.MaxAttempts),
	}); err != nil {
		return fmt.Errorf("increasing otp attempts: %w", err)
	}

	return nil
}
