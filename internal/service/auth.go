package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
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
	Login(ctx context.Context, user *domain.User) (string, string, error)
}

type authService struct {
	userRepository      repository.UserRepository
	jwtService          JwtService
	refreshTokenService RefreshTokenService
}

func NewAuthService(userRepository repository.UserRepository, jwtService JwtService, refreshTokenService RefreshTokenService) AuthService {
	return &authService{
		userRepository:      userRepository,
		jwtService:          jwtService,
		refreshTokenService: refreshTokenService,
	}
}

func (s *authService) RegisterUser(ctx context.Context, user *domain.User) error {
	user.ID = uuid.New()

	hashedPassword, err := secure.HashPassword(user.Password)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	user.Password = hashedPassword

	if err := s.userRepository.Save(ctx, user); err != nil {
		if err == repository.ErrEmailAlreadyRegistered {
			return fmt.Errorf("saving user to repository: %w", clienterrors.NewBadRequestError("there is already a user registered with this email", err))
		}

		return fmt.Errorf("saving user to repository: %w", err)
	}

	return nil
}

func (s *authService) Login(ctx context.Context, user *domain.User) (string, string, error) {
	existingUser, err := s.userRepository.GetByEmail(ctx, user.Email)
	if err != nil {
		return "", "", fmt.Errorf("fetching user by email: %w", err)
	}

	if existingUser == nil {
		return "", "", fmt.Errorf("validating user existence: %w", clienterrors.NewBadRequestError("Invalid email or password.", ErrUserNotRegistered))
	}

	if !secure.CheckPassword(existingUser.Password, user.Password) {
		return "", "", fmt.Errorf("checking password: %w", clienterrors.NewBadRequestError("Invalid email or password.", ErrInvalidCredentials))
	}

	accessToken, err := s.jwtService.GenerateAccessToken(existingUser.ID)
	if err != nil {
		return "", "", fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, err := s.refreshTokenService.GenerateRefreshToken(ctx, existingUser.ID)
	if err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}

	return accessToken, refreshToken.ID.String(), nil
}
