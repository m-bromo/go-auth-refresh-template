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

type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type userRepository struct {
	querier sqlc.Querier
}

func NewUserRepository(querier sqlc.Querier) UserRepository {
	return &userRepository{
		querier: querier,
	}
}

func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
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

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.querier.GetByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
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
