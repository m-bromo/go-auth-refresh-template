package mocks

import "context"

type OtpService struct {
	SendCodeFunc   func(ctx context.Context, email string) error
	VerifyCodeFunc func(ctx context.Context, code string, email string) error

	SendCodeCalls   int
	VerifyCodeCalls int
	LastEmail       string
	LastCode        string
}

func (m *OtpService) SendCode(ctx context.Context, email string) error {
	m.SendCodeCalls++
	m.LastEmail = email

	if m.SendCodeFunc == nil {
		return nil
	}

	return m.SendCodeFunc(ctx, email)
}

func (m *OtpService) VerifyCode(ctx context.Context, code string, email string) error {
	m.VerifyCodeCalls++
	m.LastCode = code
	m.LastEmail = email

	if m.VerifyCodeFunc == nil {
		return nil
	}

	return m.VerifyCodeFunc(ctx, code, email)
}
