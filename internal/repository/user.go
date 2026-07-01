package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

var (
	ErrEmailAlreadyRegistered = errors.New("the user email has already been registered")
)

type SqlcUserRepository struct {
	querier sqlc.Querier
}

func NewSqlcUserRepository(querier sqlc.Querier) *SqlcUserRepository {
	return &SqlcUserRepository{
		querier: querier,
	}
}

func (r *SqlcUserRepository) Save(ctx context.Context, user *domain.User) error {
	var pgErr *pq.Error

	err := r.querier.SaveUser(ctx, sqlc.SaveUserParams{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Username: user.Username,
	})
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrEmailAlreadyRegistered
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *SqlcUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := r.querier.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &domain.User{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Username: user.Username,
	}, nil
}

func (r *SqlcUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.querier.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &domain.User{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Username: user.Username,
	}, nil
}

func (r *SqlcUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, password string) error {
	if err := r.querier.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:       userID,
		Password: password,
	}); err != nil {
		return err
	}

	return nil
}
