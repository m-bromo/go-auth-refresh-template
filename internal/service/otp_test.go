package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/mocks"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/pkg/secure"
)

func TestOtpService_SendCode(t *testing.T) {
	t.Parallel()

	userRepositoryErr := errors.New("user repository failed")
	otpRepositoryErr := errors.New("otp repository failed")
	emailErr := errors.New("email failed")

	tests := []struct {
		name             string
		user             *domain.User
		getByEmailErr    error
		saveCodeErr      error
		sendCodeErr      error
		wantWrapped      string
		wantSaveCalls    int
		wantEmailCalls   int
		wantHashMatches  bool
		wantFetchedEmail string
	}{
		{
			name:             "sends code to registered user",
			user:             &domain.User{Email: "user@test.com"},
			wantSaveCalls:    1,
			wantEmailCalls:   1,
			wantHashMatches:  true,
			wantFetchedEmail: "user@test.com",
		},
		{
			name:             "does nothing for unknown user",
			wantFetchedEmail: "user@test.com",
		},
		{
			name:             "wraps user lookup error",
			getByEmailErr:    userRepositoryErr,
			wantWrapped:      "fetching user by email",
			wantFetchedEmail: "user@test.com",
		},
		{
			name:             "wraps save code error",
			user:             &domain.User{Email: "user@test.com"},
			saveCodeErr:      otpRepositoryErr,
			wantWrapped:      "saving hash code",
			wantSaveCalls:    1,
			wantFetchedEmail: "user@test.com",
		},
		{
			name:             "wraps email sender error",
			user:             &domain.User{Email: "user@test.com"},
			sendCodeErr:      emailErr,
			wantWrapped:      "sending coding",
			wantSaveCalls:    1,
			wantEmailCalls:   1,
			wantHashMatches:  true,
			wantFetchedEmail: "user@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userRepository := &mocks.UserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return tt.user, tt.getByEmailErr
				},
			}
			otpRepository := &mocks.OtpRepository{
				SaveCodeFunc: func(ctx context.Context, email string, code string) error {
					return tt.saveCodeErr
				},
			}
			emailSender := &mocks.EmailSender{
				SendCodeFunc: func(ctx context.Context, email string, code string) error {
					return tt.sendCodeErr
				},
			}
			cfg := testConfig()
			otpService := service.NewOtpService(
				otpRepository,
				userRepository,
				&mocks.ResetTokenRepository{},
				emailSender,
				&cfg.OTP,
				&cfg.ResetToken,
			)

			err := otpService.SendCode(t.Context(), "user@test.com")

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, firstNonNil(tt.getByEmailErr, tt.saveCodeErr, tt.sendCodeErr))
			} else if err != nil {
				t.Fatalf("SendCode() error = %v, want nil", err)
			}

			if userRepository.LastGetByEmail != tt.wantFetchedEmail {
				t.Errorf("GetByEmail() email = %q, want %q", userRepository.LastGetByEmail, tt.wantFetchedEmail)
			}

			if otpRepository.SaveCodeCalls != tt.wantSaveCalls {
				t.Errorf("SaveCode() calls = %d, want %d", otpRepository.SaveCodeCalls, tt.wantSaveCalls)
			}

			if emailSender.SendCodeCalls != tt.wantEmailCalls {
				t.Errorf("SendCode() email calls = %d, want %d", emailSender.SendCodeCalls, tt.wantEmailCalls)
			}

			if tt.wantHashMatches && !secure.VerifyOTP(emailSender.LastCode, otpRepository.LastSavedCode, []byte(testConfig().OTP.Secret)) {
				t.Errorf("saved OTP hash does not match emailed code")
			}
		})
	}
}

func TestOtpService_VerifyLoginCode(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository failed")
	validCode := "123456"

	tests := []struct {
		name             string
		code             string
		consumed         bool
		consumeErr       error
		wantErr          error
		wantErrCode      domain.ErrorCode
		wantWrapped      string
		wantConsumeCalls int
	}{
		{
			name:             "consumes code when it matches",
			code:             validCode,
			consumed:         true,
			wantConsumeCalls: 1,
		},
		{
			name:             "wraps repository error",
			code:             validCode,
			consumeErr:       repositoryErr,
			wantWrapped:      "consuming otp code",
			wantConsumeCalls: 1,
		},
		{
			name:             "rejects not consumed code",
			code:             "000000",
			wantErr:          service.ErrInvalidOtpCode,
			wantErrCode:      domain.ResourceNotFound,
			wantConsumeCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			otpRepository := &mocks.OtpRepository{
				ConsumeCodeIfMatchesFunc: func(ctx context.Context, email string, code string) (bool, error) {
					return tt.consumed, tt.consumeErr
				},
			}
			cfg := testConfig()
			otpService := service.NewOtpService(
				otpRepository,
				&mocks.UserRepository{},
				&mocks.ResetTokenRepository{},
				&mocks.EmailSender{},
				&cfg.OTP,
				&cfg.ResetToken,
			)

			err := otpService.VerifyLoginCode(t.Context(), tt.code, "user@test.com")

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrCode, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, tt.consumeErr)
			} else if err != nil {
				t.Fatalf("VerifyLoginCode() error = %v, want nil", err)
			}

			if otpRepository.ConsumeCodeIfMatchesCalls != tt.wantConsumeCalls {
				t.Errorf(
					"ConsumeCodeIfMatches() calls = %d, want %d",
					otpRepository.ConsumeCodeIfMatchesCalls,
					tt.wantConsumeCalls,
				)
			}

			wantConsumedCode := secure.HashOTP(tt.code, []byte(testConfig().OTP.Secret))
			if tt.wantConsumeCalls > 0 && otpRepository.LastConsumedCode != wantConsumedCode {
				t.Errorf("ConsumeCodeIfMatches() code = %q, want %q", otpRepository.LastConsumedCode, wantConsumedCode)
			}
		})
	}
}

func TestOtpService_VerifyPasswordResetCode(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository failed")
	userRepositoryErr := errors.New("user repository failed")
	saveResetTokenErr := errors.New("save reset token failed")
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	validCode := "123456"

	tests := []struct {
		name             string
		code             string
		user             *domain.User
		consumed         bool
		consumeErr       error
		getUserErr       error
		saveResetErr     error
		wantErr          error
		wantErrCode      domain.ErrorCode
		wantWrapped      string
		wantConsumeCalls int
		wantSaveCalls    int
		wantResetToken   bool
	}{
		{
			name:             "consumes code and saves reset token when it matches",
			code:             validCode,
			user:             &domain.User{ID: userID, Email: "user@test.com"},
			consumed:         true,
			wantConsumeCalls: 1,
			wantSaveCalls:    1,
			wantResetToken:   true,
		},
		{
			name:             "wraps repository error",
			code:             validCode,
			consumeErr:       repositoryErr,
			wantWrapped:      "consuming otp code",
			wantConsumeCalls: 1,
		},
		{
			name:             "rejects not consumed code",
			code:             "000000",
			wantErr:          service.ErrInvalidOtpCode,
			wantErrCode:      domain.ResourceNotFound,
			wantConsumeCalls: 1,
		},
		{
			name:             "wraps user lookup error",
			code:             validCode,
			consumed:         true,
			getUserErr:       userRepositoryErr,
			wantWrapped:      "fetching user by email",
			wantConsumeCalls: 1,
		},
		{
			name:             "rejects missing user",
			code:             validCode,
			consumed:         true,
			wantErr:          service.ErrUserNotRegistered,
			wantErrCode:      domain.Unauthenticated,
			wantConsumeCalls: 1,
		},
		{
			name:             "wraps reset token persistence error",
			code:             validCode,
			user:             &domain.User{ID: userID, Email: "user@test.com"},
			consumed:         true,
			saveResetErr:     saveResetTokenErr,
			wantWrapped:      "saving reset token",
			wantConsumeCalls: 1,
			wantSaveCalls:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			otpRepository := &mocks.OtpRepository{
				ConsumeCodeIfMatchesFunc: func(ctx context.Context, email string, code string) (bool, error) {
					return tt.consumed, tt.consumeErr
				},
			}
			userRepository := &mocks.UserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					return tt.user, tt.getUserErr
				},
			}
			resetTokenRepository := &mocks.ResetTokenRepository{
				SaveFunc: func(ctx context.Context, token *domain.ResetToken) error {
					return tt.saveResetErr
				},
			}
			cfg := testConfig()
			resetTokenExpiresFrom := time.Now().Add(cfg.ResetToken.Duration)
			otpService := service.NewOtpService(
				otpRepository,
				userRepository,
				resetTokenRepository,
				&mocks.EmailSender{},
				&cfg.OTP,
				&cfg.ResetToken,
			)

			resetToken, err := otpService.VerifyPasswordResetCode(t.Context(), tt.code, "user@test.com")
			resetTokenExpiresUntil := time.Now().Add(cfg.ResetToken.Duration)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrCode, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(
					t,
					err,
					tt.wantWrapped,
					firstNonNil(tt.consumeErr, tt.getUserErr, tt.saveResetErr),
				)
			} else if err != nil {
				t.Fatalf("VerifyPasswordResetCode() error = %v, want nil", err)
			}

			if otpRepository.ConsumeCodeIfMatchesCalls != tt.wantConsumeCalls {
				t.Errorf(
					"ConsumeCodeIfMatches() calls = %d, want %d",
					otpRepository.ConsumeCodeIfMatchesCalls,
					tt.wantConsumeCalls,
				)
			}

			wantConsumedCode := secure.HashOTP(tt.code, []byte(testConfig().OTP.Secret))
			if tt.wantConsumeCalls > 0 && otpRepository.LastConsumedCode != wantConsumedCode {
				t.Errorf("ConsumeCodeIfMatches() code = %q, want %q", otpRepository.LastConsumedCode, wantConsumedCode)
			}

			if resetTokenRepository.SaveCalls != tt.wantSaveCalls {
				t.Errorf("Save() reset token calls = %d, want %d", resetTokenRepository.SaveCalls, tt.wantSaveCalls)
			}

			if tt.wantResetToken {
				if resetToken == "" {
					t.Fatalf("VerifyPasswordResetCode() reset token is empty")
				}

				if resetTokenRepository.LastSavedToken == nil {
					t.Fatalf("saved reset token is nil")
				}

				if resetTokenRepository.LastSavedToken.UserID != userID {
					t.Errorf("saved reset token user ID = %s, want %s", resetTokenRepository.LastSavedToken.UserID, userID)
				}

				if resetTokenRepository.LastSavedToken.ExpiresAt.IsZero() {
					t.Errorf("saved reset token expiration is zero")
				}

				if resetTokenRepository.LastSavedToken.ExpiresAt.Before(resetTokenExpiresFrom) ||
					resetTokenRepository.LastSavedToken.ExpiresAt.After(resetTokenExpiresUntil) {
					t.Errorf(
						"saved reset token expiration = %s, want between %s and %s",
						resetTokenRepository.LastSavedToken.ExpiresAt,
						resetTokenExpiresFrom,
						resetTokenExpiresUntil,
					)
				}

				if resetTokenRepository.LastSavedToken.TokenHash == "" {
					t.Fatalf("saved reset token is empty")
				}

				if resetTokenRepository.LastSavedToken.TokenHash == resetToken {
					t.Errorf("saved reset token should be hashed, got raw returned token")
				}

				if !secure.VerifyResetToken(
					resetToken,
					resetTokenRepository.LastSavedToken.TokenHash,
					[]byte(cfg.ResetToken.Secret),
				) {
					t.Errorf("saved reset token hash does not match returned token")
				}
			}
		})
	}
}

func testConfig() *configs.Config {
	return &configs.Config{
		Jwt: configs.Jwt{
			PrivateKey: "test-secret",
			Duration:   15 * time.Minute,
		},
		RefreshToken: configs.RefreshToken{
			Duration: 24 * time.Hour,
		},
		OTP: configs.OTP{
			MaxValue: 1000000,
			Secret:   "otp-secret",
			Duration: 2 * time.Minute,
		},
		ResetToken: configs.ResetToken{
			Secret:   "reset-token-secret",
			Duration: 10 * time.Minute,
		},
	}
}
