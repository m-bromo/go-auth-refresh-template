package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrUserNotFound  = errors.New("user not found")
)

type UserByIDFinder interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

type UserService interface {
	GetProfile(ctx context.Context, id string) (*domain.User, error)
}

type userService struct {
	userFinder UserByIDFinder
}

func NewUserService(userFinder UserByIDFinder) UserService {
	return &userService{
		userFinder: userFinder,
	}
}

func (s *userService) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.NewBadRequestError("invalid user id", ErrInvalidUserID)
	}

	user, err := s.userFinder.GetByID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("fetching user from repository by ID: %w", err)
	}

	if user == nil {
		return nil, domain.NewNotFoundError("user not found", ErrUserNotFound)
	}

	return user, nil
}
