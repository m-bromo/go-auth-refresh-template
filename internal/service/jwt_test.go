package service_test

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/service"
)

func TestJwtService_GenerateAccessToken(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	jwtService := service.NewJwtService(testConfig())

	tokenString, err := jwtService.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v, want nil", err)
	}

	claims, err := jwtService.ValidateAccessToken("Bearer " + tokenString)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v, want nil", err)
	}

	if claims.Subject != userID.String() {
		t.Errorf("claims subject = %q, want %q", claims.Subject, userID.String())
	}

	if claims.ID == "" {
		t.Errorf("claims ID = empty, want generated ID")
	}

	if claims.ExpiresAt == nil {
		t.Fatalf("claims ExpiresAt = nil, want expiration")
	}
}

func TestJwtService_ValidateAccessToken(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	jwtService := service.NewJwtService(testConfig())
	validToken, err := jwtService.GenerateAccessToken(userID)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: userID.String(),
	})
	wrongSecretTokenString, err := wrongSecretToken.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	noneTokenString, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject: userID.String(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString() none token error = %v", err)
	}

	tests := []struct {
		name        string
		bearerToken string
		wantSubject string
		wantErr     error
		wantErrType domain.ErrorType
		wantWrapped string
	}{
		{
			name:        "accepts bearer token",
			bearerToken: "Bearer " + validToken,
			wantSubject: userID.String(),
		},
		{
			name:        "rejects empty token",
			bearerToken: "Bearer ",
			wantErr:     service.ErrTokenNotProvided,
			wantErrType: domain.Unauthorized,
		},
		{
			name:        "rejects raw token format",
			bearerToken: validToken,
			wantErr:     service.ErrInvalidToken,
			wantErrType: domain.Unauthorized,
		},
		{
			name:        "rejects malformed token",
			bearerToken: "Bearer not-a-jwt",
			wantErr:     service.ErrInvalidToken,
			wantErrType: domain.Unauthorized,
		},
		{
			name:        "rejects wrong signature",
			bearerToken: "Bearer " + wrongSecretTokenString,
			wantErr:     service.ErrInvalidToken,
			wantErrType: domain.Unauthorized,
		},
		{
			name:        "wraps invalid signing method",
			bearerToken: "Bearer " + noneTokenString,
			wantWrapped: "parsing access token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claims, err := jwtService.ValidateAccessToken(tt.bearerToken)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
				return
			}

			if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, nil)

				if !errors.Is(err, service.ErrInvalidSigningMethod) {
					t.Fatalf("error does not wrap ErrInvalidSigningMethod: %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("ValidateAccessToken() error = %v, want nil", err)
			}

			if claims.Subject != tt.wantSubject {
				t.Errorf("claims subject = %q, want %q", claims.Subject, tt.wantSubject)
			}
		})
	}
}
