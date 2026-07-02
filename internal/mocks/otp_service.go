package mocks

import (
	"context"

	"github.com/google/uuid"
)

type OtpService struct {
	SendCodeFunc                func(ctx context.Context, email string) (uuid.UUID, error)
	VerifyLoginCodeFunc         func(ctx context.Context, code string, email string) error
	VerifyPasswordResetCodeFunc func(ctx context.Context, code string, challengeID uuid.UUID) (string, error)

	SendCodeCalls                int
	VerifyLoginCodeCalls         int
	VerifyPasswordResetCodeCalls int
	LastEmail                    string
	LastCode                     string
	LastChallengeID              uuid.UUID
}

func (m *OtpService) SendCode(ctx context.Context, email string) (uuid.UUID, error) {
	m.SendCodeCalls++
	m.LastEmail = email

	if m.SendCodeFunc == nil {
		return uuid.Nil, nil
	}

	return m.SendCodeFunc(ctx, email)
}

func (m *OtpService) VerifyLoginCode(ctx context.Context, code string, email string) error {
	m.VerifyLoginCodeCalls++
	m.LastCode = code
	m.LastEmail = email

	if m.VerifyLoginCodeFunc == nil {
		return nil
	}

	return m.VerifyLoginCodeFunc(ctx, code, email)
}

func (m *OtpService) VerifyPasswordResetCode(
	ctx context.Context,
	code string,
	challengeID uuid.UUID,
) (string, error) {
	m.VerifyPasswordResetCodeCalls++
	m.LastCode = code
	m.LastChallengeID = challengeID

	if m.VerifyPasswordResetCodeFunc == nil {
		return "", nil
	}

	return m.VerifyPasswordResetCodeFunc(ctx, code, challengeID)
}
