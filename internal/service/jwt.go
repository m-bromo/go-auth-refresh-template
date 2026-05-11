package service

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/config"
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
	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Subject: userID.String(),
		ID:      uuid.NewString(),
	}).SignedString(s.cfg.Jwt.PrivateKey)
	if err != nil {
		return "", nil
	}

	return token, nil
}

func (s *jwtService) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, ErrInvalidSigningMethod
		}

		return s.cfg.Jwt.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, err
}
