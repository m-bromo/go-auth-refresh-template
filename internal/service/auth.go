package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	apierrors "github.com/m-bromo/go-auth-template/internal/api_errors"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

var (
	ErrUserAlreadyRegistered = errors.New("this user's email was already registered")
	ErrUserNotRegistered     = errors.New("this user is not registered")
	ErrInvalidCredentials    = errors.New("the user has invalid credentials")
)

type AuthService interface {
	RegisterUser(ctx context.Context, user *domain.User) error
	Login(ctx context.Context, user *domain.User) (*domain.User, error)
}

type authService struct {
	userRepository repository.UserRepository
}

func NewAuthService(userRepository repository.UserRepository) AuthService {
	return &authService{
		userRepository: userRepository,
	}
}

func (a *authService) RegisterUser(ctx context.Context, user *domain.User) error {
	user.ID = uuid.New()

	hashedPassword, err := secure.HashPassword(user.Password)
	if err != nil {
		return fmt.Errorf("register user: %w", apierrors.NewInternalServerError(err.Error()))
	}
	user.Password = hashedPassword

	if err := a.userRepository.Save(ctx, user); err != nil {
		return fmt.Errorf("register user: %w", apierrors.NewInternalServerError(err.Error()))
	}

	return nil
}

func (a *authService) Login(ctx context.Context, user *domain.User) (*domain.User, error) {
	existingUser, err := a.userRepository.GetByEmail(ctx, user.Email)
	if err != nil {
		return nil, fmt.Errorf("login: %w", apierrors.NewInternalServerError(err.Error()))
	}

	if existingUser == nil {
		return nil, fmt.Errorf("login: %w", apierrors.NewBadRequestError(ErrUserNotRegistered.Error()))
	}

	if !secure.CheckPassword(existingUser.Password, user.Password) {
		return nil, fmt.Errorf("login: %w", apierrors.NewBadRequestError(ErrInvalidCredentials.Error()))
	}

	return existingUser, nil
}
