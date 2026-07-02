package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/pkg/secure"
)

var (
	ErrOtpCodeNotFound = errors.New("otp code not found in database")
	ErrInvalidOtpCode  = errors.New("the otp code does not match")
)

type OTPRepository interface {
	Consume(ctx context.Context, email string, code string) (*domain.OTP, error)
	ConsumeByChallengeID(ctx context.Context, challengeID uuid.UUID, code string) (*domain.OTP, error)
	IncreaseAttempts(ctx context.Context, challengeID uuid.UUID) error
}

type OTPUserFinder interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type ResetTokenSaver interface {
	Save(ctx context.Context, token *domain.ResetToken) error
}

type EmailSender interface {
	SendCode(ctx context.Context, email string, code string) error
}

type OtpService interface {
	SendCode(ctx context.Context, email string) (uuid.UUID, error)
	VerifyLoginCode(ctx context.Context, code string, email string) error
	VerifyPasswordResetCode(ctx context.Context, code string, challengeID uuid.UUID) (string, error)
}

type otpService struct {
	otpRepository     OTPRepository
	unitOfWork        repository.UnitOfWork
	otpUserFinder     OTPUserFinder
	resetTokenSaver   ResetTokenSaver
	emailSender       EmailSender
	otpOptions        *configs.OTP
	resetTokenOptions *configs.ResetToken
}

func NewOtpService(
	otpRepository OTPRepository,
	unitOfWork repository.UnitOfWork,
	otpUserFinder OTPUserFinder,
	resetTokenSaver ResetTokenSaver,
	emailSender EmailSender,
	otpOptions *configs.OTP,
	resetTokenOptions *configs.ResetToken,
) OtpService {
	return &otpService{
		otpRepository:     otpRepository,
		unitOfWork:        unitOfWork,
		otpUserFinder:     otpUserFinder,
		resetTokenSaver:   resetTokenSaver,
		emailSender:       emailSender,
		otpOptions:        otpOptions,
		resetTokenOptions: resetTokenOptions,
	}
}

func (s *otpService) SendCode(ctx context.Context, email string) (uuid.UUID, error) {
	challengeID := uuid.New()

	user, err := s.otpUserFinder.GetByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("fetching user by email: %w", err)
	}

	if user == nil {
		return challengeID, nil
	}

	code, err := rand.Int(rand.Reader, big.NewInt(int64(s.otpOptions.MaxValue)))
	if err != nil {
		return uuid.Nil, fmt.Errorf("generating otp code: %w", err)
	}

	formatedCode := fmt.Sprintf("%06d", code)

	hashedCode := secure.HashOTP(formatedCode, []byte(s.otpOptions.Secret))

	if err := s.emailSender.SendCode(ctx, user.Email, formatedCode); err != nil {
		return uuid.Nil, fmt.Errorf("sending coding: %w", err)
	}

	otp := domain.OTP{
		ID:         challengeID,
		Identifier: user.Email,
		Code:       hashedCode,
		Attempts:   0,
		ExpiresAt:  time.Now().Add(s.otpOptions.Duration),
		CreatedAt:  time.Now(),
	}

	if err := s.unitOfWork.Exec(ctx, func(repos repository.Repositories) error {
		if err := repos.OTPRepository.InvalidateByIdentifier(ctx, user.Email); err != nil {
			return fmt.Errorf("invalidating previous otp codes: %w", err)
		}

		if err := repos.OTPRepository.Save(ctx, &otp); err != nil {
			return fmt.Errorf("saving hash code: %w", err)
		}

		return nil
	}); err != nil {
		return uuid.Nil, fmt.Errorf("replacing otp code: %w", err)
	}

	return challengeID, nil
}

func (s *otpService) VerifyLoginCode(ctx context.Context, code string, email string) error {
	hashedCode := secure.HashOTP(code, []byte(s.otpOptions.Secret))
	otp, err := s.otpRepository.Consume(ctx, email, hashedCode)
	if err != nil {
		return fmt.Errorf("consuming otp code: %w", err)
	}

	if otp == nil {
		return domain.NewResourceNotFoundError("the inserted code does not match", ErrInvalidOtpCode)
	}

	return nil
}

func (s *otpService) VerifyPasswordResetCode(
	ctx context.Context,
	code string,
	challengeID uuid.UUID,
) (string, error) {
	hashedCode := secure.HashOTP(code, []byte(s.otpOptions.Secret))
	otp, err := s.otpRepository.ConsumeByChallengeID(ctx, challengeID, hashedCode)
	if err != nil {
		return "", fmt.Errorf("consuming otp code: %w", err)
	}

	if otp == nil {
		if err := s.otpRepository.IncreaseAttempts(ctx, challengeID); err != nil {
			return "", fmt.Errorf("increasing otp attempts: %w", err)
		}

		return "", domain.NewResourceNotFoundError("the inserted code does not match", ErrInvalidOtpCode)
	}

	user, err := s.otpUserFinder.GetByEmail(ctx, otp.Identifier)
	if err != nil {
		return "", fmt.Errorf("fetching user by email: %w", err)
	}

	if user == nil {
		return "", domain.NewPermissionDeniedError("invalid credentials")
	}

	resetToken, err := secure.GenerateResetToken()
	if err != nil {
		return "", fmt.Errorf("generating reset token: %w", err)
	}

	hashedResetToken := secure.HashResetToken(resetToken, []byte(s.resetTokenOptions.Secret))

	token := domain.ResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashedResetToken,
		ExpiresAt: time.Now().Add(s.resetTokenOptions.Duration),
		CreatedAt: time.Now(),
	}

	if err := s.resetTokenSaver.Save(ctx, &token); err != nil {
		return "", fmt.Errorf("saving reset token: %w", err)
	}

	return resetToken, nil
}
