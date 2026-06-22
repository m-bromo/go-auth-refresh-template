package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type RefreshTokenRepository struct {
	SaveFunc           func(ctx context.Context, token *domain.RefreshToken) error
	GetFunc            func(ctx context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error)
	ConsumeFunc        func(ctx context.Context, tokenID uuid.UUID) (string, error)
	DeleteFunc         func(ctx context.Context, tokenID uuid.UUID) error
	DeleteByUserIDFunc func(ctx context.Context, userID uuid.UUID) error

	SaveCalls           int
	GetCalls            int
	ConsumeCalls        int
	DeleteCalls         int
	DeleteByUserIDCalls int
	LastSaved           *domain.RefreshToken
	LastGotID           uuid.UUID
	LastConsumed        uuid.UUID
	LastDeleted         uuid.UUID
	LastDeletedUserID   uuid.UUID
}

func (m *RefreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	m.SaveCalls++
	tokenCopy := *token
	m.LastSaved = &tokenCopy

	if m.SaveFunc == nil {
		return nil
	}

	return m.SaveFunc(ctx, token)
}

func (m *RefreshTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (*domain.RefreshToken, error) {
	m.GetCalls++
	m.LastGotID = tokenID

	if m.GetFunc == nil {
		return nil, nil
	}

	return m.GetFunc(ctx, tokenID)
}

func (m *RefreshTokenRepository) Consume(ctx context.Context, tokenID uuid.UUID) (string, error) {
	m.ConsumeCalls++
	m.LastConsumed = tokenID

	if m.ConsumeFunc == nil {
		return "", nil
	}

	return m.ConsumeFunc(ctx, tokenID)
}

func (m *RefreshTokenRepository) Delete(ctx context.Context, tokenID uuid.UUID) error {
	m.DeleteCalls++
	m.LastDeleted = tokenID

	if m.DeleteFunc == nil {
		return nil
	}

	return m.DeleteFunc(ctx, tokenID)
}

func (m *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	m.DeleteByUserIDCalls++
	m.LastDeletedUserID = userID

	if m.DeleteByUserIDFunc == nil {
		return nil
	}

	return m.DeleteByUserIDFunc(ctx, userID)
}
