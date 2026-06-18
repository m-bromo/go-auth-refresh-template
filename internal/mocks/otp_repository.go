package mocks

import "context"

type OtpRepository struct {
	SaveCodeFunc       func(ctx context.Context, email string, code string) error
	DeleteCodeFunc     func(ctx context.Context, email string) error
	GetCodeByEmailFunc func(ctx context.Context, email string) (string, error)

	SaveCodeCalls       int
	DeleteCodeCalls     int
	GetCodeByEmailCalls int
	LastSavedEmail      string
	LastSavedCode       string
	LastDeletedEmail    string
	LastFetchedEmail    string
}

func (m *OtpRepository) SaveCode(ctx context.Context, email string, code string) error {
	m.SaveCodeCalls++
	m.LastSavedEmail = email
	m.LastSavedCode = code

	if m.SaveCodeFunc == nil {
		return nil
	}

	return m.SaveCodeFunc(ctx, email, code)
}

func (m *OtpRepository) DeleteCode(ctx context.Context, email string) error {
	m.DeleteCodeCalls++
	m.LastDeletedEmail = email

	if m.DeleteCodeFunc == nil {
		return nil
	}

	return m.DeleteCodeFunc(ctx, email)
}

func (m *OtpRepository) GetCodeByEmail(ctx context.Context, email string) (string, error) {
	m.GetCodeByEmailCalls++
	m.LastFetchedEmail = email

	if m.GetCodeByEmailFunc == nil {
		return "", nil
	}

	return m.GetCodeByEmailFunc(ctx, email)
}
