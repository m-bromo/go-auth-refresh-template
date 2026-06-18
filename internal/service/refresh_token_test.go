package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/mocks"
	"github.com/m-bromo/go-auth-template/internal/service"
)

func TestRefreshTokenService_GenerateRefreshToken(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repositoryErr := errors.New("repository failed")

	tests := []struct {
		name        string
		saveErr     error
		wantWrapped string
	}{
		{
			name: "generates and stores token",
		},
		{
			name:        "wraps repository error",
			saveErr:     repositoryErr,
			wantWrapped: "saving refresh token to repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			refreshTokenRepository := &mocks.RefreshTokenRepository{
				SaveFunc: func(ctx context.Context, token *domain.RefreshToken) error {
					return tt.saveErr
				},
			}
			refreshTokenService := service.NewRefreshTokenService(
				refreshTokenRepository,
				&mocks.JwtService{},
			)

			token, err := refreshTokenService.GenerateRefreshToken(t.Context(), userID)

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, repositoryErr)
				return
			}

			if err != nil {
				t.Fatalf("GenerateRefreshToken() error = %v, want nil", err)
			}

			if token.ID == uuid.Nil {
				t.Errorf("token ID = nil, want generated UUID")
			}

			if token.UserID != userID {
				t.Errorf("token user ID = %s, want %s", token.UserID, userID)
			}

			if refreshTokenRepository.SaveCalls != 1 {
				t.Fatalf("Save() calls = %d, want 1", refreshTokenRepository.SaveCalls)
			}

			if refreshTokenRepository.LastSaved.UserID != userID {
				t.Errorf("saved token user ID = %s, want %s", refreshTokenRepository.LastSaved.UserID, userID)
			}
		})
	}
}

func TestRefreshTokenService_Refresh(t *testing.T) {
	t.Parallel()

	tokenID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	consumeErr := errors.New("consume failed")
	saveErr := errors.New("save failed")
	jwtErr := errors.New("jwt failed")

	tests := []struct {
		name                string
		tokenIDString       string
		consumedUserID      string
		consumeErr          error
		saveErr             error
		generateTokenErr    error
		wantAccessToken     string
		wantNewRefreshToken bool
		wantErr             error
		wantErrType         domain.ErrorType
		wantWrapped         string
		wantConsumeCalls    int
		wantSaveCalls       int
		wantJwtCalls        int
	}{
		{
			name:                "rotates refresh token and returns access token",
			tokenIDString:       tokenID.String(),
			consumedUserID:      userID.String(),
			wantAccessToken:     "new-access-token",
			wantNewRefreshToken: true,
			wantConsumeCalls:    1,
			wantSaveCalls:       1,
			wantJwtCalls:        1,
		},
		{
			name:          "rejects malformed refresh token",
			tokenIDString: "not-a-uuid",
			wantErr:       service.ErrInvalidRefreshToken,
			wantErrType:   domain.Unauthorized,
		},
		{
			name:             "wraps consume error",
			tokenIDString:    tokenID.String(),
			consumeErr:       consumeErr,
			wantWrapped:      "fetching refresh token from repository",
			wantConsumeCalls: 1,
		},
		{
			name:             "rejects missing refresh token",
			tokenIDString:    tokenID.String(),
			wantErr:          service.ErrRefreshTokenNotFoundOrExpired,
			wantErrType:      domain.Unauthorized,
			wantConsumeCalls: 1,
		},
		{
			name:             "wraps malformed stored user id",
			tokenIDString:    tokenID.String(),
			consumedUserID:   "not-a-uuid",
			wantWrapped:      "parsing user id",
			wantConsumeCalls: 1,
		},
		{
			name:             "wraps save error",
			tokenIDString:    tokenID.String(),
			consumedUserID:   userID.String(),
			saveErr:          saveErr,
			wantWrapped:      "saving new refresh token to repository",
			wantConsumeCalls: 1,
			wantSaveCalls:    1,
		},
		{
			name:             "wraps access token generation error",
			tokenIDString:    tokenID.String(),
			consumedUserID:   userID.String(),
			generateTokenErr: jwtErr,
			wantWrapped:      "generating new access token",
			wantConsumeCalls: 1,
			wantSaveCalls:    1,
			wantJwtCalls:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			refreshTokenRepository := &mocks.RefreshTokenRepository{
				ConsumeFunc: func(ctx context.Context, tokenID uuid.UUID) (string, error) {
					return tt.consumedUserID, tt.consumeErr
				},
				SaveFunc: func(ctx context.Context, token *domain.RefreshToken) error {
					return tt.saveErr
				},
			}
			jwtService := &mocks.JwtService{
				GenerateAccessTokenFunc: func(userID uuid.UUID) (string, error) {
					return "new-access-token", tt.generateTokenErr
				},
			}
			refreshTokenService := service.NewRefreshTokenService(
				refreshTokenRepository,
				jwtService,
			)

			accessToken, newRefreshToken, err := refreshTokenService.Refresh(t.Context(), tt.tokenIDString)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, firstNonNil(tt.consumeErr, tt.saveErr, tt.generateTokenErr))
			} else if err != nil {
				t.Fatalf("Refresh() error = %v, want nil", err)
			}

			if accessToken != tt.wantAccessToken {
				t.Errorf("accessToken = %q, want %q", accessToken, tt.wantAccessToken)
			}

			if tt.wantNewRefreshToken {
				if _, err := uuid.Parse(newRefreshToken); err != nil {
					t.Fatalf("new refresh token = %q, want UUID: %v", newRefreshToken, err)
				}

				if newRefreshToken == tt.tokenIDString {
					t.Errorf("new refresh token reused old token ID")
				}
			}

			if refreshTokenRepository.ConsumeCalls != tt.wantConsumeCalls {
				t.Errorf("Consume() calls = %d, want %d", refreshTokenRepository.ConsumeCalls, tt.wantConsumeCalls)
			}

			if refreshTokenRepository.SaveCalls != tt.wantSaveCalls {
				t.Errorf("Save() calls = %d, want %d", refreshTokenRepository.SaveCalls, tt.wantSaveCalls)
			}

			if jwtService.GenerateAccessTokenCalls != tt.wantJwtCalls {
				t.Errorf("GenerateAccessToken() calls = %d, want %d", jwtService.GenerateAccessTokenCalls, tt.wantJwtCalls)
			}
		})
	}
}

func TestRefreshTokenService_Revoke(t *testing.T) {
	t.Parallel()

	tokenID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	deleteErr := errors.New("delete failed")

	tests := []struct {
		name            string
		tokenIDString   string
		deleteErr       error
		wantErr         error
		wantErrType     domain.ErrorType
		wantWrapped     string
		wantDeleteCalls int
	}{
		{
			name:            "deletes refresh token",
			tokenIDString:   tokenID.String(),
			wantDeleteCalls: 1,
		},
		{
			name:          "rejects malformed refresh token",
			tokenIDString: "not-a-uuid",
			wantErr:       service.ErrInvalidRefreshToken,
			wantErrType:   domain.Unauthorized,
		},
		{
			name:            "wraps delete error",
			tokenIDString:   tokenID.String(),
			deleteErr:       deleteErr,
			wantWrapped:     "deleting refresh token",
			wantDeleteCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			refreshTokenRepository := &mocks.RefreshTokenRepository{
				DeleteFunc: func(ctx context.Context, tokenID uuid.UUID) error {
					return tt.deleteErr
				},
			}
			refreshTokenService := service.NewRefreshTokenService(
				refreshTokenRepository,
				&mocks.JwtService{},
			)

			err := refreshTokenService.Revoke(t.Context(), tt.tokenIDString)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, deleteErr)
			} else if err != nil {
				t.Fatalf("Revoke() error = %v, want nil", err)
			}

			if refreshTokenRepository.DeleteCalls != tt.wantDeleteCalls {
				t.Errorf("Delete() calls = %d, want %d", refreshTokenRepository.DeleteCalls, tt.wantDeleteCalls)
			}
		})
	}
}
