package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type RefreshTokenRepository struct {
	SaveFunc    func(ctx context.Context, token *domain.RefreshToken) error
	GetFunc     func(ctx context.Context, tokenID uuid.UUID) (string, error)
	ConsumeFunc func(ctx context.Context, tokenID uuid.UUID) (string, error)
	DeleteFunc  func(ctx context.Context, tokenID uuid.UUID) error

	SaveCalls    int
	GetCalls     int
	ConsumeCalls int
	DeleteCalls  int
	LastSaved    *domain.RefreshToken
	LastGotID    uuid.UUID
	LastConsumed uuid.UUID
	LastDeleted  uuid.UUID
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

func (m *RefreshTokenRepository) Get(ctx context.Context, tokenID uuid.UUID) (string, error) {
	m.GetCalls++
	m.LastGotID = tokenID

	if m.GetFunc == nil {
		return "", nil
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
