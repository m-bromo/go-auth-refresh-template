package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

var (
	ErrUserAlreadyRegistered = errors.New("user email is already registered")
	ErrUserNotRegistered     = errors.New("user is not registered")
	ErrInvalidCredentials    = errors.New("invalid user credentials")
)

type AuthService interface {
	RegisterUser(ctx context.Context, user *domain.User) error
	Login(ctx context.Context, user *domain.User) (string, string, error)
	LoginWithOtp(ctx context.Context, email string, code string) (string, string, error)
}

type authService struct {
	userRepository      repository.UserRepository
	jwtService          JwtService
	refreshTokenService RefreshTokenService
	otpService          OtpService
}

func NewAuthService(
	userRepository repository.UserRepository,
	jwtService JwtService,
	refreshTokenService RefreshTokenService,
	otpService OtpService,
) AuthService {
	return &authService{
		userRepository:      userRepository,
		jwtService:          jwtService,
		refreshTokenService: refreshTokenService,
		otpService:          otpService,
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
		if errors.Is(err, repository.ErrEmailAlreadyRegistered) {
			return domain.NewConflictError("user email is already registered", ErrUserAlreadyRegistered)
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
		return "", "", domain.NewUnauthorizedError("invalid email or password", ErrUserNotRegistered)
	}

	if !secure.CheckPassword(existingUser.Password, user.Password) {
		return "", "", domain.NewUnauthorizedError("invalid email or password", ErrInvalidCredentials)
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

func (s *authService) LoginWithOtp(ctx context.Context, email string, code string) (string, string, error) {
	existingUser, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("fetching user by email: %w", err)
	}

	if existingUser == nil {
		return "", "", domain.NewUnauthorizedError("invalid email or otp code", ErrUserNotRegistered)
	}

	if err := s.otpService.VerifyCode(ctx, code, email); err != nil {
		return "", "", fmt.Errorf("verifying otp code: %w", err)
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
