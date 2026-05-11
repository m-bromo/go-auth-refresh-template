package repository

import (
	"context"
	"database/sql"

	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
)

type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
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
	return r.querier.SaveUser(ctx, sqlc.SaveUserParams{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Username: user.Username,
	})
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := r.querier.GetByEmail(ctx, email)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &domain.User{
		ID:       user.ID,
		Email:    user.Email,
		Password: user.Password,
		Username: user.Password,
	}, nil
}
