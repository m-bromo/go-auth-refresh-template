package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/email"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

var (
	ErrOtpCodeNotFound = errors.New("otp code not found in database")
	ErrInvalidOtpCode  = errors.New("the otp code does not match")
)

type OtpService interface {
	SendCode(ctx context.Context, email string) error
	VerifyLoginCode(ctx context.Context, code string, email string) error
	VerifyPasswordResetCode(ctx context.Context, code string, email string) (string, error)
}

type otpService struct {
	otpRepository        repository.OtpRepository
	userRepository       repository.UserRepository
	resetTokenRepository repository.ResetTokenRepository
	emailSender          email.EmailSender
	cfg                  *config.Config
}

func NewOtpService(
	otpRepository repository.OtpRepository,
	userRepository repository.UserRepository,
	resetTokenRepository repository.ResetTokenRepository,
	emailSender email.EmailSender,
	cfg *config.Config,
) OtpService {
	return &otpService{
		otpRepository:        otpRepository,
		userRepository:       userRepository,
		resetTokenRepository: resetTokenRepository,
		emailSender:          emailSender,
		cfg:                  cfg,
	}
}

func (s *otpService) SendCode(ctx context.Context, email string) error {
	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("fetching user by email: %w", err)
	}

	if user == nil {
		return nil
	}

	code, err := rand.Int(rand.Reader, big.NewInt(int64(s.cfg.OTP.MaxValue)))
	if err != nil {
		return fmt.Errorf("generating otp code: %w", err)
	}

	formatedCode := fmt.Sprintf("%06d", code)

	hashedCode := secure.HashOTP(formatedCode, []byte(s.cfg.OTP.Secret))

	if err := s.otpRepository.SaveCode(ctx, user.Email, hashedCode); err != nil {
		return fmt.Errorf("saving hash code: %w", err)
	}

	if err := s.emailSender.SendCode(ctx, user.Email, formatedCode); err != nil {
		return fmt.Errorf("sending coding: %w", err)
	}

	return nil
}

func (s *otpService) VerifyLoginCode(ctx context.Context, code string, email string) error {
	hashedCode := secure.HashOTP(code, []byte(s.cfg.OTP.Secret))
	consumed, err := s.otpRepository.ConsumeCodeIfMatches(ctx, email, hashedCode)
	if err != nil {
		return fmt.Errorf("consuming otp code: %w", err)
	}

	if !consumed {
		return domain.NewNotFoundError("the inserted code does not match", ErrInvalidOtpCode)
	}

	return nil
}

func (s *otpService) VerifyPasswordResetCode(ctx context.Context, code string, email string) (string, error) {
	hashedCode := secure.HashOTP(code, []byte(s.cfg.OTP.Secret))
	consumed, err := s.otpRepository.ConsumeCodeIfMatches(ctx, email, hashedCode)
	if err != nil {
		return "", fmt.Errorf("consuming otp code: %w", err)
	}

	if !consumed {
		return "", domain.NewNotFoundError("the inserted code does not match", ErrInvalidOtpCode)
	}

	user, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("fetching user by email: %w", err)
	}

	if user == nil {
		return "", domain.NewUnauthorizedError("invalid email or otp code", ErrUserNotRegistered)
	}

	resetToken, err := secure.GenerateResetToken()
	if err != nil {
		return "", fmt.Errorf("generating reset token: %w", err)
	}

	hashedResetToken := secure.HashResetToken(resetToken, []byte(s.cfg.ResetToken.Secret))

	token := domain.ResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashedResetToken,
		ExpiresAt: time.Now().Add(s.cfg.ResetToken.Duration),
		CreatedAt: time.Now(),
	}

	if err := s.resetTokenRepository.Save(ctx, &token); err != nil {
		return "", fmt.Errorf("saving reset token: %w", err)
	}

	return resetToken, nil
}
