package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type ResetTokenRepository struct {
	SaveFunc            func(ctx context.Context, token *domain.ResetToken) error
	GetValidByHashFunc  func(ctx context.Context, tokenHash string) (*domain.ResetToken, error)
	ConsumeFunc         func(ctx context.Context, tokenHash string) (*domain.ResetToken, error)
	DeleteByUserIDFunc  func(ctx context.Context, userID uuid.UUID) error
	SaveCalls           int
	GetValidByHashCalls int
	ConsumeCalls        int
	DeleteByUserIDCalls int
	LastSavedToken      *domain.ResetToken
	LastTokenHash       string
	LastDeletedUserID   uuid.UUID
}

func (m *ResetTokenRepository) Save(ctx context.Context, token *domain.ResetToken) error {
	m.SaveCalls++
	tokenCopy := *token
	m.LastSavedToken = &tokenCopy

	if m.SaveFunc == nil {
		return nil
	}

	return m.SaveFunc(ctx, token)
}

func (m *ResetTokenRepository) GetValidByHash(ctx context.Context, tokenHash string) (*domain.ResetToken, error) {
	m.GetValidByHashCalls++
	m.LastTokenHash = tokenHash

	if m.GetValidByHashFunc == nil {
		return nil, nil
	}

	return m.GetValidByHashFunc(ctx, tokenHash)
}

func (m *ResetTokenRepository) Consume(ctx context.Context, tokenHash string) (*domain.ResetToken, error) {
	m.ConsumeCalls++
	m.LastTokenHash = tokenHash

	if m.ConsumeFunc == nil {
		return nil, nil
	}

	return m.ConsumeFunc(ctx, tokenHash)
}

func (m *ResetTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	m.DeleteByUserIDCalls++
	m.LastDeletedUserID = userID

	if m.DeleteByUserIDFunc == nil {
		return nil
	}

	return m.DeleteByUserIDFunc(ctx, userID)
}
