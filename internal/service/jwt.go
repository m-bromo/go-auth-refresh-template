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
	ErrInvalidSigningMethod = errors.New("invalid token signing method")
	ErrInvalidClaims        = errors.New("invalid token claims")
	ErrTokenNotProvided     = errors.New("token string was not provided")
	ErrInvalidToken         = errors.New("invalid token")
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
		return nil, clienterrors.NewUnauthorizedError("token string was not provided", ErrTokenNotProvided)
	}

	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, clienterrors.NewUnauthorizedError("invalid token signing method", ErrInvalidSigningMethod)
		}

		return []byte(s.cfg.Jwt.PrivateKey), nil
	})
	if err != nil {
		return nil, clienterrors.NewUnauthorizedError("invalid token", ErrInvalidToken)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, clienterrors.NewUnauthorizedError("invalid token claims", ErrInvalidClaims)
	}

	return claims, nil
}
