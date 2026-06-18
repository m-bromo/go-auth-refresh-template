package mocks

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtService struct {
	GenerateAccessTokenFunc func(userID uuid.UUID) (string, error)
	ValidateAccessTokenFunc func(tokenString string) (*jwt.RegisteredClaims, error)

	GenerateAccessTokenCalls int
	ValidateAccessTokenCalls int
	LastGenerateUserID       uuid.UUID
	LastValidateTokenString  string
}

func (m *JwtService) GenerateAccessToken(userID uuid.UUID) (string, error) {
	m.GenerateAccessTokenCalls++
	m.LastGenerateUserID = userID

	if m.GenerateAccessTokenFunc == nil {
		return "", nil
	}

	return m.GenerateAccessTokenFunc(userID)
}

func (m *JwtService) ValidateAccessToken(tokenString string) (*jwt.RegisteredClaims, error) {
	m.ValidateAccessTokenCalls++
	m.LastValidateTokenString = tokenString

	if m.ValidateAccessTokenFunc == nil {
		return nil, nil
	}

	return m.ValidateAccessTokenFunc(tokenString)
}
