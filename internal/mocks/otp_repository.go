package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
)

type OtpRepository struct {
	SaveFunc                   func(ctx context.Context, otp *domain.OTP) error
	InvalidateByIdentifierFunc func(ctx context.Context, identifier string) error
	ConsumeFunc                func(ctx context.Context, email string, code string) (*domain.OTP, error)
	ConsumeByChallengeIDFunc   func(ctx context.Context, challengeID uuid.UUID, code string) (*domain.OTP, error)
	IncreaseAttemptsFunc       func(ctx context.Context, challengeID uuid.UUID) error

	SaveCalls                   int
	InvalidateByIdentifierCalls int
	ConsumeCalls                int
	ConsumeByChallengeIDCalls   int
	IncreaseAttemptsCalls       int
	LastSavedOTP                *domain.OTP
	LastInvalidatedIdentifier   string
	LastConsumedEmail           string
	LastConsumedChallengeID     uuid.UUID
	LastConsumedCode            string
	LastIncreasedChallengeID    uuid.UUID
}

func (m *OtpRepository) InvalidateByIdentifier(ctx context.Context, identifier string) error {
	m.InvalidateByIdentifierCalls++
	m.LastInvalidatedIdentifier = identifier

	if m.InvalidateByIdentifierFunc == nil {
		return nil
	}

	return m.InvalidateByIdentifierFunc(ctx, identifier)
}

func (m *OtpRepository) Save(ctx context.Context, otp *domain.OTP) error {
	m.SaveCalls++
	otpCopy := *otp
	m.LastSavedOTP = &otpCopy

	if m.SaveFunc == nil {
		return nil
	}

	return m.SaveFunc(ctx, otp)
}

func (m *OtpRepository) Consume(ctx context.Context, email string, code string) (*domain.OTP, error) {
	m.ConsumeCalls++
	m.LastConsumedEmail = email
	m.LastConsumedCode = code

	if m.ConsumeFunc == nil {
		return nil, nil
	}

	return m.ConsumeFunc(ctx, email, code)
}

func (m *OtpRepository) ConsumeByChallengeID(
	ctx context.Context,
	challengeID uuid.UUID,
	code string,
) (*domain.OTP, error) {
	m.ConsumeByChallengeIDCalls++
	m.LastConsumedChallengeID = challengeID
	m.LastConsumedCode = code

	if m.ConsumeByChallengeIDFunc == nil {
		return nil, nil
	}

	return m.ConsumeByChallengeIDFunc(ctx, challengeID, code)
}

func (m *OtpRepository) IncreaseAttempts(ctx context.Context, challengeID uuid.UUID) error {
	m.IncreaseAttemptsCalls++
	m.LastIncreasedChallengeID = challengeID

	if m.IncreaseAttemptsFunc == nil {
		return nil
	}

	return m.IncreaseAttemptsFunc(ctx, challengeID)
}
