package mocks

import "context"

type OtpRepository struct {
	SaveCodeFunc             func(ctx context.Context, email string, code string) error
	ConsumeCodeIfMatchesFunc func(ctx context.Context, email string, code string) (bool, error)

	SaveCodeCalls             int
	ConsumeCodeIfMatchesCalls int
	LastSavedEmail            string
	LastSavedCode             string
	LastConsumedEmail         string
	LastConsumedCode          string
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

func (m *OtpRepository) ConsumeCodeIfMatches(ctx context.Context, email string, code string) (bool, error) {
	m.ConsumeCodeIfMatchesCalls++
	m.LastConsumedEmail = email
	m.LastConsumedCode = code

	if m.ConsumeCodeIfMatchesFunc == nil {
		return false, nil
	}

	return m.ConsumeCodeIfMatchesFunc(ctx, email, code)
}
