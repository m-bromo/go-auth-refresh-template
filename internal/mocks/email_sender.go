package mocks

import "context"

type EmailSender struct {
	SendCodeFunc func(ctx context.Context, email string, code string) error

	SendCodeCalls int
	LastEmail     string
	LastCode      string
}

func (m *EmailSender) SendCode(ctx context.Context, email string, code string) error {
	m.SendCodeCalls++
	m.LastEmail = email
	m.LastCode = code

	if m.SendCodeFunc == nil {
		return nil
	}

	return m.SendCodeFunc(ctx, email, code)
}
