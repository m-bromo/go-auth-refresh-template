package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/config"
	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
)

var (
	ErrInvalidSigningMethod = errors.New("the token signing method is invalid")
	ErrInvalidClaims        = errors.New("the token claims are invalid")
	ErrTokenNotProvided     = errors.New("token string was not provided")
)

type JwtService interface {
	GenerateAccessToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(tokenString string) (*jwt.RegisteredClaims, error)
}

type jwtService struct {
	cfg *config.Config
}

func NewJwtService(cfg *config.Config) JwtService {
	return &jwtService{
		cfg: cfg,
	}
}

func (s *jwtService) GenerateAccessToken(userID uuid.UUID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID.String(),
		ID:        uuid.NewString(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.Jwt.Duration)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})

	tokenString, err := token.SignedString([]byte(s.cfg.Jwt.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("signing access token: %w", err)
	}

	return tokenString, nil
}

func (s *jwtService) ValidateAccessToken(bearerToken string) (*jwt.RegisteredClaims, error) {
	tokenString := strings.TrimPrefix(bearerToken, "Bearer ")

	if tokenString == "" {
		return nil, fmt.Errorf("verifying token string format: %w", clienterrors.NewUnauthorizedError("failed to validadate token format", ErrTokenNotProvided))
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("verifying signing method: %w", clienterrors.NewUnauthorizedError("failed to validate signing method", ErrInvalidSigningMethod))
		}

		return []byte(s.cfg.Jwt.PrivateKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token with claims: %w", clienterrors.NewUnauthorizedError("failed to parse token", err))
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("validating token claims: %w", clienterrors.NewUnauthorizedError("failed to validate claims", ErrInvalidClaims))
	}

	return claims, nil
}
