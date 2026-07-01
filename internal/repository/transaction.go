package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type UnitOfWork interface {
	Exec(ctx context.Context, fn func(repos Repositories) error) error
}

type Repositories struct {
	UserRepository         UserRepository
	ResetTokenRepository   ResetTokenRepository
	RefreshTokenRepository RefreshTokenRepository
}

type unitOfWork struct {
	cfg     *configs.Config
	db      *sql.DB
	queries *sqlc.Queries
}

func NewUnitOfWork(
	cfg *configs.Config,
	db *sql.DB,
	queries *sqlc.Queries,
) UnitOfWork {
	return &unitOfWork{
		cfg:     cfg,
		db:      db,
		queries: queries,
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
		RefreshTokenRepository: NewRefreshTokenRepository(qtx, u.cfg),
		UserRepository:         NewUserRepository(qtx),
		ResetTokenRepository:   NewResetTokenRepository(qtx),
	}

	if err := fn(repos); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
