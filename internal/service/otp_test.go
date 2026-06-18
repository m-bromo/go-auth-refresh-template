package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/mocks"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/service"
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
			otpService := service.NewOtpService(
				otpRepository,
				userRepository,
				emailSender,
				testConfig(),
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

func TestOtpService_VerifyCode(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository failed")
	deleteErr := errors.New("delete failed")
	validCode := "123456"
	hashedCode := secure.HashOTP(validCode, []byte(testConfig().OTP.Secret))

	tests := []struct {
		name            string
		code            string
		foundCode       string
		getCodeErr      error
		deleteCodeErr   error
		wantErr         error
		wantErrType     domain.ErrorType
		wantWrapped     string
		wantDeleteCalls int
	}{
		{
			name:            "deletes code when it matches",
			code:            validCode,
			foundCode:       hashedCode,
			wantDeleteCalls: 1,
		},
		{
			name:        "wraps repository error",
			code:        validCode,
			getCodeErr:  repositoryErr,
			wantWrapped: "getting otp code",
		},
		{
			name:        "rejects missing code",
			code:        validCode,
			wantErr:     service.ErrOtpCodeNotFound,
			wantErrType: domain.BadRequest,
		},
		{
			name:        "rejects invalid code",
			code:        "000000",
			foundCode:   hashedCode,
			wantErr:     service.ErrInvalidOtpCode,
			wantErrType: domain.NotFound,
		},
		{
			name:            "wraps delete error",
			code:            validCode,
			foundCode:       hashedCode,
			deleteCodeErr:   deleteErr,
			wantWrapped:     "deleting otp code",
			wantDeleteCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			otpRepository := &mocks.OtpRepository{
				GetCodeByEmailFunc: func(ctx context.Context, email string) (string, error) {
					return tt.foundCode, tt.getCodeErr
				},
				DeleteCodeFunc: func(ctx context.Context, email string) error {
					return tt.deleteCodeErr
				},
			}
			otpService := service.NewOtpService(
				otpRepository,
				&mocks.UserRepository{},
				&mocks.EmailSender{},
				testConfig(),
			)

			err := otpService.VerifyCode(t.Context(), tt.code, "user@test.com")

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, firstNonNil(tt.getCodeErr, tt.deleteCodeErr))
			} else if err != nil {
				t.Fatalf("VerifyCode() error = %v, want nil", err)
			}

			if otpRepository.DeleteCodeCalls != tt.wantDeleteCalls {
				t.Errorf("DeleteCode() calls = %d, want %d", otpRepository.DeleteCodeCalls, tt.wantDeleteCalls)
			}
		})
	}
}

func testConfig() *config.Config {
	return &config.Config{
		Jwt: config.Jwt{
			PrivateKey: "test-secret",
			Duration:   15 * time.Minute,
		},
		OTP: config.OTP{
			MaxValue: 1000000,
			Secret:   "otp-secret",
			Duration: 2 * time.Minute,
		},
	}
}
