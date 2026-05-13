package service

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/config"
	apierrors "github.com/m-bromo/go-auth-template/internal/api_errors"
)

var (
	ErrInvalidSigningMethod = errors.New("the token signing method is invalid")
	ErrInvalidClaims        = errors.New("the token claims are invalid")
)

type JwtService interface {
	GenerateToken(userID uuid.UUID) (string, error)
	ValidateToken(tokenString string) (*jwt.RegisteredClaims, error)
}

type jwtService struct {
	cfg *config.Config
}

func NewJwtService(cfg *config.Config) JwtService {
	return &jwtService{
		cfg: cfg,
	}
}

func (s *jwtService) GenerateToken(userID uuid.UUID) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: userID.String(),
		ID:      uuid.NewString(),
	})

	tokenString, err := token.SignedString([]byte(s.cfg.Jwt.PrivateKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("validate token: %w", apierrors.NewUnauthorizedError(ErrInvalidSigningMethod.Error()))
		}

		return []byte(s.cfg.Jwt.PublicKey), nil
	})
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", apierrors.NewInternalServerError(err.Error()))
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("validate token : %w", apierrors.NewUnauthorizedError(ErrInvalidClaims.Error()))
	}

	return claims, nil
}
