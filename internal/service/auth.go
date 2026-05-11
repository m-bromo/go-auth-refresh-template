package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
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
		return err
	}
	user.Password = hashedPassword

	if err := a.userRepository.Save(ctx, user); err != nil {
		return err
	}

	return nil
}

func (a *authService) Login(ctx context.Context, user *domain.User) (*domain.User, error) {
	existingUser, err := a.userRepository.GetByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if existingUser == nil {
		return nil, ErrUserNotRegistered
	}

	if !secure.CheckPassword(user.Password, existingUser.Password) {
		return nil, ErrInvalidCredentials
	}

	return existingUser, nil
}
