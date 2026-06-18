package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/mocks"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
)

func TestAuthService_RegisterUser(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository failed")

	tests := []struct {
		name          string
		saveErr       error
		wantErr       error
		wantErrType   domain.ErrorType
		wantWrapped   string
		wantSavedUser bool
	}{
		{
			name:          "registers user with hashed password",
			wantSavedUser: true,
		},
		{
			name:        "maps duplicated email to conflict domain error",
			saveErr:     repository.ErrEmailAlreadyRegistered,
			wantErr:     service.ErrUserAlreadyRegistered,
			wantErrType: domain.Conflict,
		},
		{
			name:        "wraps repository error",
			saveErr:     repositoryErr,
			wantWrapped: "saving user to repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userRepository := &mocks.UserRepository{
				SaveFunc: func(ctx context.Context, user *domain.User) error {
					return tt.saveErr
				},
			}
			authService := service.NewAuthService(
				userRepository,
				&mocks.JwtService{},
				&mocks.RefreshTokenService{},
				&mocks.OtpService{},
			)
			user := &domain.User{
				Email:    "user@test.com",
				Username: "user",
				Password: "password@123",
			}

			err := authService.RegisterUser(t.Context(), user)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
				return
			}

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, repositoryErr)
				return
			}

			if err != nil {
				t.Fatalf("RegisterUser() error = %v, want nil", err)
			}

			if userRepository.SaveCalls != 1 {
				t.Fatalf("Save() calls = %d, want 1", userRepository.SaveCalls)
			}

			if !tt.wantSavedUser {
				return
			}

			if userRepository.LastSavedUser.ID == uuid.Nil {
				t.Errorf("saved user ID = nil, want generated UUID")
			}

			if userRepository.LastSavedUser.Password == "password@123" {
				t.Errorf("saved password was not hashed")
			}

			if !secure.CheckPassword(userRepository.LastSavedUser.Password, "password@123") {
				t.Errorf("saved password hash does not match original password")
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	refreshTokenID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repositoryErr := errors.New("repository failed")
	jwtErr := errors.New("jwt failed")
	refreshErr := errors.New("refresh token failed")
	hashedPassword, err := secure.HashPassword("password@123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name             string
		inputPassword    string
		existingUser     *domain.User
		getByEmailErr    error
		generateTokenErr error
		refreshTokenErr  error
		wantAccessToken  string
		wantRefreshToken string
		wantErr          error
		wantErrType      domain.ErrorType
		wantWrapped      string
	}{
		{
			name:          "returns access and refresh tokens",
			inputPassword: "password@123",
			existingUser: &domain.User{
				ID:       userID,
				Email:    "user@test.com",
				Password: hashedPassword,
			},
			wantAccessToken:  "access-token",
			wantRefreshToken: refreshTokenID.String(),
		},
		{
			name:          "wraps repository error",
			getByEmailErr: repositoryErr,
			wantWrapped:   "fetching user by email",
		},
		{
			name:          "rejects missing user",
			inputPassword: "password@123",
			wantErr:       service.ErrUserNotRegistered,
			wantErrType:   domain.Unauthorized,
		},
		{
			name:          "rejects invalid password",
			inputPassword: "wrong-password",
			existingUser: &domain.User{
				ID:       userID,
				Email:    "user@test.com",
				Password: hashedPassword,
			},
			wantErr:     service.ErrInvalidCredentials,
			wantErrType: domain.Unauthorized,
		},
		{
			name:          "wraps access token generation error",
			inputPassword: "password@123",
			existingUser: &domain.User{
				ID:       userID,
				Email:    "user@test.com",
				Password: hashedPassword,
			},
			generateTokenErr: jwtErr,
			wantWrapped:      "generating access token",
		},
		{
			name:          "wraps refresh token generation error",
			inputPassword: "password@123",
			existingUser: &domain.User{
				ID:       userID,
				Email:    "user@test.com",
				Password: hashedPassword,
			},
			refreshTokenErr: refreshErr,
			wantWrapped:     "generating refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userRepository := &mocks.UserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return tt.existingUser, tt.getByEmailErr
				},
			}
			jwtService := &mocks.JwtService{
				GenerateAccessTokenFunc: func(userID uuid.UUID) (string, error) {
					return "access-token", tt.generateTokenErr
				},
			}
			refreshTokenService := &mocks.RefreshTokenService{
				GenerateRefreshTokenFunc: func(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
					return &domain.RefreshToken{
						ID:     refreshTokenID,
						UserID: userID,
					}, tt.refreshTokenErr
				},
			}
			authService := service.NewAuthService(
				userRepository,
				jwtService,
				refreshTokenService,
				&mocks.OtpService{},
			)

			accessToken, refreshToken, err := authService.Login(t.Context(), &domain.User{
				Email:    "user@test.com",
				Password: tt.inputPassword,
			})

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
				return
			}

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, firstNonNil(tt.getByEmailErr, tt.generateTokenErr, tt.refreshTokenErr))
				return
			}

			if err != nil {
				t.Fatalf("Login() error = %v, want nil", err)
			}

			if accessToken != tt.wantAccessToken {
				t.Errorf("accessToken = %q, want %q", accessToken, tt.wantAccessToken)
			}

			if refreshToken != tt.wantRefreshToken {
				t.Errorf("refreshToken = %q, want %q", refreshToken, tt.wantRefreshToken)
			}
		})
	}
}

func TestAuthService_LoginWithOtp(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	refreshTokenID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repositoryErr := errors.New("repository failed")
	otpErr := errors.New("otp failed")
	jwtErr := errors.New("jwt failed")
	refreshErr := errors.New("refresh token failed")

	tests := []struct {
		name             string
		existingUser     *domain.User
		getByEmailErr    error
		verifyCodeErr    error
		generateTokenErr error
		refreshTokenErr  error
		wantAccessToken  string
		wantRefreshToken string
		wantErr          error
		wantErrType      domain.ErrorType
		wantWrapped      string
	}{
		{
			name: "returns access and refresh tokens",
			existingUser: &domain.User{
				ID:    userID,
				Email: "user@test.com",
			},
			wantAccessToken:  "access-token",
			wantRefreshToken: refreshTokenID.String(),
		},
		{
			name:          "wraps repository error",
			getByEmailErr: repositoryErr,
			wantWrapped:   "fetching user by email",
		},
		{
			name:        "rejects missing user",
			wantErr:     service.ErrUserNotRegistered,
			wantErrType: domain.Unauthorized,
		},
		{
			name: "wraps otp verification error",
			existingUser: &domain.User{
				ID:    userID,
				Email: "user@test.com",
			},
			verifyCodeErr: otpErr,
			wantWrapped:   "verifying otp code",
		},
		{
			name: "wraps access token generation error",
			existingUser: &domain.User{
				ID:    userID,
				Email: "user@test.com",
			},
			generateTokenErr: jwtErr,
			wantWrapped:      "generating access token",
		},
		{
			name: "wraps refresh token generation error",
			existingUser: &domain.User{
				ID:    userID,
				Email: "user@test.com",
			},
			refreshTokenErr: refreshErr,
			wantWrapped:     "generating refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userRepository := &mocks.UserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return tt.existingUser, tt.getByEmailErr
				},
			}
			otpService := &mocks.OtpService{
				VerifyCodeFunc: func(ctx context.Context, code string, email string) error {
					return tt.verifyCodeErr
				},
			}
			jwtService := &mocks.JwtService{
				GenerateAccessTokenFunc: func(userID uuid.UUID) (string, error) {
					return "access-token", tt.generateTokenErr
				},
			}
			refreshTokenService := &mocks.RefreshTokenService{
				GenerateRefreshTokenFunc: func(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
					return &domain.RefreshToken{
						ID:     refreshTokenID,
						UserID: userID,
					}, tt.refreshTokenErr
				},
			}
			authService := service.NewAuthService(
				userRepository,
				jwtService,
				refreshTokenService,
				otpService,
			)

			accessToken, refreshToken, err := authService.LoginWithOtp(t.Context(), "user@test.com", "123456")

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
				return
			}

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, firstNonNil(
					tt.getByEmailErr,
					tt.verifyCodeErr,
					tt.generateTokenErr,
					tt.refreshTokenErr,
				))
				return
			}

			if err != nil {
				t.Fatalf("LoginWithOtp() error = %v, want nil", err)
			}

			if accessToken != tt.wantAccessToken {
				t.Errorf("accessToken = %q, want %q", accessToken, tt.wantAccessToken)
			}

			if refreshToken != tt.wantRefreshToken {
				t.Errorf("refreshToken = %q, want %q", refreshToken, tt.wantRefreshToken)
			}
		})
	}
}

func assertDomainError(t *testing.T, err error, wantType domain.ErrorType, wantErr error) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want domain error")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("error = %T, want *domain.DomainError", err)
	}

	if domainErr.ErrorType != wantType {
		t.Fatalf("domain error type = %q, want %q", domainErr.ErrorType, wantType)
	}

	if !errors.Is(domainErr, wantErr) {
		t.Fatalf("domain error does not wrap %v: %v", wantErr, domainErr.Err)
	}
}

func assertWrappedError(t *testing.T, err error, wantMessage string, wantErr error) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want wrapped error")
	}

	if !strings.Contains(err.Error(), wantMessage) {
		t.Fatalf("error = %q, want message containing %q", err.Error(), wantMessage)
	}

	if wantErr != nil && !errors.Is(err, wantErr) {
		t.Fatalf("error does not wrap %v: %v", wantErr, err)
	}
}

func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
