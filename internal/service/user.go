package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apierrors "github.com/m-bromo/go-auth-template/internal/api_errors"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

var (
	ErrUserNotFound = errors.New("the user was not found in database")
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
		return nil, fmt.Errorf("parsing user ID: %w", apierrors.NewBadRequestError("invalid uuid format", err))
	}

	user, err := s.userRepository.GetByID(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("fetching user from repository by ID: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("validating user existence: %w", apierrors.NewNotFoundError("user does not exists", ErrUserNotFound))
	}

	return user, nil
}
