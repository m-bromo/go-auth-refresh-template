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
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/pkg/secure"
)

func TestOtpService_SendCode(t *testing.T) {
	t.Parallel()

	userRepositoryErr := errors.New("user repository failed")
	otpRepositoryErr := errors.New("otp repository failed")
	invalidateErr := errors.New("invalidate otp failed")
	emailErr := errors.New("email failed")

	tests := []struct {
		name                string
		user                *domain.User
		getByEmailErr       error
		invalidateErr       error
		saveCodeErr         error
		sendCodeErr         error
		wantWrapped         string
		wantUoWCalls        int
		wantInvalidateCalls int
		wantSaveCalls       int
		wantEmailCalls      int
		wantHashMatches     bool
		wantChallengeID     bool
		wantFetchedEmail    string
	}{
		{
			name:                "sends code to registered user",
			user:                &domain.User{Email: "user@test.com"},
			wantSaveCalls:       1,
			wantUoWCalls:        1,
			wantInvalidateCalls: 1,
			wantEmailCalls:      1,
			wantHashMatches:     true,
			wantChallengeID:     true,
			wantFetchedEmail:    "user@test.com",
		},
		{
			name:             "does nothing for unknown user",
			wantChallengeID:  true,
			wantFetchedEmail: "user@test.com",
		},
		{
			name:             "wraps user lookup error",
			getByEmailErr:    userRepositoryErr,
			wantWrapped:      "fetching user by email",
			wantFetchedEmail: "user@test.com",
		},
		{
			name:                "wraps invalidate code error",
			user:                &domain.User{Email: "user@test.com"},
			invalidateErr:       invalidateErr,
			wantWrapped:         "invalidating previous otp codes",
			wantUoWCalls:        1,
			wantInvalidateCalls: 1,
			wantEmailCalls:      1,
			wantFetchedEmail:    "user@test.com",
		},
		{
			name:                "wraps save code error",
			user:                &domain.User{Email: "user@test.com"},
			saveCodeErr:         otpRepositoryErr,
			wantWrapped:         "saving hash code",
			wantUoWCalls:        1,
			wantInvalidateCalls: 1,
			wantSaveCalls:       1,
			wantEmailCalls:      1,
			wantFetchedEmail:    "user@test.com",
		},
		{
			name:             "wraps email sender error",
			user:             &domain.User{Email: "user@test.com"},
			sendCodeErr:      emailErr,
			wantWrapped:      "sending coding",
			wantEmailCalls:   1,
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
				InvalidateByIdentifierFunc: func(ctx context.Context, identifier string) error {
					return tt.invalidateErr
				},
				SaveFunc: func(ctx context.Context, otp *domain.OTP) error {
					return tt.saveCodeErr
				},
			}
			unitOfWork := &mocks.UnitOfWork{
				Repos: repository.Repositories{
					OTPRepository: otpRepository,
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
				unitOfWork,
				userRepository,
				&mocks.ResetTokenRepository{},
				emailSender,
				&cfg.OTP,
				&cfg.ResetToken,
			)

			challengeID, err := otpService.SendCode(t.Context(), "user@test.com")

			if tt.wantWrapped != "" {
				assertWrappedError(
					t,
					err,
					tt.wantWrapped,
					firstNonNil(tt.getByEmailErr, tt.invalidateErr, tt.saveCodeErr, tt.sendCodeErr),
				)
			} else if err != nil {
				t.Fatalf("SendCode() error = %v, want nil", err)
			}

			if tt.wantChallengeID && challengeID == uuid.Nil {
				t.Errorf("SendCode() challenge ID = nil, want generated UUID")
			}

			if !tt.wantChallengeID && challengeID != uuid.Nil {
				t.Errorf("SendCode() challenge ID = %s, want nil UUID", challengeID)
			}

			if userRepository.LastGetByEmail != tt.wantFetchedEmail {
				t.Errorf("GetByEmail() email = %q, want %q", userRepository.LastGetByEmail, tt.wantFetchedEmail)
			}

			if unitOfWork.ExecCalls != tt.wantUoWCalls {
				t.Errorf("UnitOfWork.Exec() calls = %d, want %d", unitOfWork.ExecCalls, tt.wantUoWCalls)
			}

			if otpRepository.InvalidateByIdentifierCalls != tt.wantInvalidateCalls {
				t.Errorf(
					"InvalidateByIdentifier() calls = %d, want %d",
					otpRepository.InvalidateByIdentifierCalls,
					tt.wantInvalidateCalls,
				)
			}

			if tt.wantInvalidateCalls > 0 && otpRepository.LastInvalidatedIdentifier != "user@test.com" {
				t.Errorf(
					"InvalidateByIdentifier() identifier = %q, want %q",
					otpRepository.LastInvalidatedIdentifier,
					"user@test.com",
				)
			}

			if otpRepository.SaveCalls != tt.wantSaveCalls {
				t.Errorf("Save() calls = %d, want %d", otpRepository.SaveCalls, tt.wantSaveCalls)
			}

			if emailSender.SendCodeCalls != tt.wantEmailCalls {
				t.Errorf("SendCode() email calls = %d, want %d", emailSender.SendCodeCalls, tt.wantEmailCalls)
			}

			if tt.wantHashMatches {
				if otpRepository.LastSavedOTP == nil {
					t.Fatal("Save() OTP = nil, want saved OTP")
				}

				if otpRepository.LastSavedOTP.ID != challengeID {
					t.Errorf("Save() OTP ID = %s, want challenge ID %s", otpRepository.LastSavedOTP.ID, challengeID)
				}

				if !secure.VerifyOTP(
					emailSender.LastCode,
					otpRepository.LastSavedOTP.Code,
					[]byte(testConfig().OTP.Secret),
				) {
					t.Errorf("saved OTP hash does not match emailed code")
				}
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
				ConsumeFunc: func(ctx context.Context, email string, code string) (*domain.OTP, error) {
					if !tt.consumed {
						return nil, tt.consumeErr
					}

					return &domain.OTP{Identifier: email}, tt.consumeErr
				},
			}
			cfg := testConfig()
			otpService := service.NewOtpService(
				otpRepository,
				&mocks.UnitOfWork{},
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

			if otpRepository.ConsumeCalls != tt.wantConsumeCalls {
				t.Errorf(
					"Consume() calls = %d, want %d",
					otpRepository.ConsumeCalls,
					tt.wantConsumeCalls,
				)
			}

			wantConsumedCode := secure.HashOTP(tt.code, []byte(testConfig().OTP.Secret))
			if tt.wantConsumeCalls > 0 && otpRepository.LastConsumedCode != wantConsumedCode {
				t.Errorf("Consume() code = %q, want %q", otpRepository.LastConsumedCode, wantConsumedCode)
			}
		})
	}
}

func TestOtpService_VerifyPasswordResetCode(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository failed")
	increaseAttemptsErr := errors.New("increase attempts failed")
	userRepositoryErr := errors.New("user repository failed")
	saveResetTokenErr := errors.New("save reset token failed")
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	challengeID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	validCode := "123456"

	tests := []struct {
		name              string
		code              string
		user              *domain.User
		consumed          bool
		consumeErr        error
		increaseErr       error
		getUserErr        error
		saveResetErr      error
		wantErr           error
		wantErrCode       domain.ErrorCode
		wantWrapped       string
		wantConsumeCalls  int
		wantIncreaseCalls int
		wantSaveCalls     int
		wantResetToken    bool
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
			name:              "rejects not consumed code",
			code:              "000000",
			wantErr:           service.ErrInvalidOtpCode,
			wantErrCode:       domain.ResourceNotFound,
			wantConsumeCalls:  1,
			wantIncreaseCalls: 1,
		},
		{
			name:              "wraps attempt increment error",
			code:              "000000",
			increaseErr:       increaseAttemptsErr,
			wantWrapped:       "increasing otp attempts",
			wantConsumeCalls:  1,
			wantIncreaseCalls: 1,
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
				ConsumeByChallengeIDFunc: func(
					ctx context.Context,
					gotChallengeID uuid.UUID,
					code string,
				) (*domain.OTP, error) {
					if !tt.consumed {
						return nil, tt.consumeErr
					}

					return &domain.OTP{
						ID:         gotChallengeID,
						Identifier: "user@test.com",
					}, tt.consumeErr
				},
				IncreaseAttemptsFunc: func(ctx context.Context, gotChallengeID uuid.UUID) error {
					return tt.increaseErr
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
				&mocks.UnitOfWork{},
				userRepository,
				resetTokenRepository,
				&mocks.EmailSender{},
				&cfg.OTP,
				&cfg.ResetToken,
			)

			resetToken, err := otpService.VerifyPasswordResetCode(t.Context(), tt.code, challengeID)
			resetTokenExpiresUntil := time.Now().Add(cfg.ResetToken.Duration)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrCode, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(
					t,
					err,
					tt.wantWrapped,
					firstNonNil(tt.consumeErr, tt.increaseErr, tt.getUserErr, tt.saveResetErr),
				)
			} else if err != nil {
				t.Fatalf("VerifyPasswordResetCode() error = %v, want nil", err)
			}

			if otpRepository.ConsumeByChallengeIDCalls != tt.wantConsumeCalls {
				t.Errorf(
					"ConsumeByChallengeID() calls = %d, want %d",
					otpRepository.ConsumeByChallengeIDCalls,
					tt.wantConsumeCalls,
				)
			}

			if otpRepository.LastConsumedChallengeID != challengeID {
				t.Errorf(
					"ConsumeByChallengeID() challenge ID = %s, want %s",
					otpRepository.LastConsumedChallengeID,
					challengeID,
				)
			}

			wantConsumedCode := secure.HashOTP(tt.code, []byte(testConfig().OTP.Secret))
			if tt.wantConsumeCalls > 0 && otpRepository.LastConsumedCode != wantConsumedCode {
				t.Errorf("ConsumeByChallengeID() code = %q, want %q", otpRepository.LastConsumedCode, wantConsumedCode)
			}

			if otpRepository.IncreaseAttemptsCalls != tt.wantIncreaseCalls {
				t.Errorf(
					"IncreaseAttempts() calls = %d, want %d",
					otpRepository.IncreaseAttemptsCalls,
					tt.wantIncreaseCalls,
				)
			}

			if tt.wantIncreaseCalls > 0 && otpRepository.LastIncreasedChallengeID != challengeID {
				t.Errorf(
					"IncreaseAttempts() challenge ID = %s, want %s",
					otpRepository.LastIncreasedChallengeID,
					challengeID,
				)
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
