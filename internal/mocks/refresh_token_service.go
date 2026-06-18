package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type RefreshTokenService struct {
	GenerateRefreshTokenFunc func(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error)
	RefreshFunc              func(ctx context.Context, tokenIDString string) (string, string, error)
	RevokeFunc               func(ctx context.Context, tokenIDString string) error

	GenerateRefreshTokenCalls int
	RefreshCalls              int
	RevokeCalls               int
	LastGenerateUserID        uuid.UUID
	LastRefreshTokenIDString  string
	LastRevokeTokenIDString   string
}

func (m *RefreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	m.GenerateRefreshTokenCalls++
	m.LastGenerateUserID = userID

	if m.GenerateRefreshTokenFunc == nil {
		return nil, nil
	}

	return m.GenerateRefreshTokenFunc(ctx, userID)
}

func (m *RefreshTokenService) Refresh(ctx context.Context, tokenIDString string) (string, string, error) {
	m.RefreshCalls++
	m.LastRefreshTokenIDString = tokenIDString

	if m.RefreshFunc == nil {
		return "", "", nil
	}

	return m.RefreshFunc(ctx, tokenIDString)
}

func (m *RefreshTokenService) Revoke(ctx context.Context, tokenIDString string) error {
	m.RevokeCalls++
	m.LastRevokeTokenIDString = tokenIDString

	if m.RevokeFunc == nil {
		return nil
	}

	return m.RevokeFunc(ctx, tokenIDString)
}
