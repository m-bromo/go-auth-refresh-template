package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

var (
	ErrInvalidUserID = errors.New("invalid user id")
	ErrUserNotFound  = errors.New("user not found")
)

type UserService interface {
	GetProfile(ctx context.Context, id string) (*domain.User, error)
}

type userService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) UserService {
	return &userService{
		userRepository: userRepository,
	}
}

func (s *userService) GetProfile(ctx context.Context, id string) (*domain.User, error) {
	uuid, err := uuid.Parse(id)
	if err != nil {
		return nil, clienterrors.NewBadRequestError("invalid user id", ErrInvalidUserID)
	}

	user, err := s.userRepository.GetByID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("fetching user from repository by ID: %w", err)
	}

	if user == nil {
		return nil, clienterrors.NewNotFoundError("user not found", ErrUserNotFound)
	}

	return user, nil
}
