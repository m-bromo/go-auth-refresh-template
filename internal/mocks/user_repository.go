package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type UserRepository struct {
	SaveFunc           func(ctx context.Context, user *domain.User) error
	GetByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmailFunc     func(ctx context.Context, email string) (*domain.User, error)
	UpdatePasswordFunc func(ctx context.Context, userID uuid.UUID, password string) error

	SaveCalls           int
	GetByIDCalls        int
	GetByEmailCalls     int
	UpdatePasswordCalls int
	LastSavedUser       *domain.User
	LastGetByID         uuid.UUID
	LastGetByEmail      string
	LastUpdatedUserID   uuid.UUID
	LastUpdatedPassword string
}

func (m *UserRepository) Save(ctx context.Context, user *domain.User) error {
	m.SaveCalls++
	userCopy := *user
	m.LastSavedUser = &userCopy

	if m.SaveFunc == nil {
		return nil
	}

	return m.SaveFunc(ctx, user)
}

func (m *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	m.GetByIDCalls++
	m.LastGetByID = id

	if m.GetByIDFunc == nil {
		return nil, nil
	}

	return m.GetByIDFunc(ctx, id)
}

func (m *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.GetByEmailCalls++
	m.LastGetByEmail = email

	if m.GetByEmailFunc == nil {
		return nil, nil
	}

	return m.GetByEmailFunc(ctx, email)
}

func (m *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, password string) error {
	m.UpdatePasswordCalls++
	m.LastUpdatedUserID = userID
	m.LastUpdatedPassword = password

	if m.UpdatePasswordFunc == nil {
		return nil
	}

	return m.UpdatePasswordFunc(ctx, userID, password)
}
