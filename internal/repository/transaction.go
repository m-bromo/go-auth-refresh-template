package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type UnitOfWork interface {
	Exec(ctx context.Context, fn func(repos Repositories) error) error
}

type UserTransactionRepository interface {
	UpdatePassword(ctx context.Context, userID uuid.UUID, password string) error
}

type ResetTokenTransactionRepository interface {
	Consume(ctx context.Context, tokenHash string) (*domain.ResetToken, error)
}

type RefreshTokenTransactionRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	Consume(ctx context.Context, tokenID uuid.UUID) (string, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type OTPTransactionRepository interface {
	InvalidateByIdentifier(ctx context.Context, identifier string) error
	Save(ctx context.Context, otp *domain.OTP) error
}

type Repositories struct {
	UserRepository         UserTransactionRepository
	ResetTokenRepository   ResetTokenTransactionRepository
	RefreshTokenRepository RefreshTokenTransactionRepository
	OTPRepository          OTPTransactionRepository
}

type unitOfWork struct {
	db         *sql.DB
	queries    *sqlc.Queries
	otpOptions *configs.OTP
}

func NewUnitOfWork(
	db *sql.DB,
	queries *sqlc.Queries,
	otpOptions *configs.OTP,
) UnitOfWork {
	return &unitOfWork{
		db:         db,
		queries:    queries,
		otpOptions: otpOptions,
	}
}

func (u *unitOfWork) Exec(ctx context.Context, fn func(repos Repositories) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := u.queries.WithTx(tx)

	repos := Repositories{
		RefreshTokenRepository: NewSqlcRefreshTokenRepository(qtx),
		UserRepository:         NewSqlcUserRepository(qtx),
		ResetTokenRepository:   NewSqlcResetTokenRepository(qtx),
		OTPRepository:          NewSqlcOtpRepository(qtx, u.otpOptions),
	}

	if err := fn(repos); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
