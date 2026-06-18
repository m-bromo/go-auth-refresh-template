package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

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
	VerifyCode(ctx context.Context, code string, email string) error
}

type otpService struct {
	otpRepository  repository.OtpRepository
	userRepository repository.UserRepository
	emailSender    email.EmailSender
	cfg            *config.Config
}

func NewOtpServic(
	otpRepository repository.OtpRepository,
	userRepository repository.UserRepository,
	emailSender email.EmailSender,
	cfg *config.Config,
) OtpService {
	return &otpService{
		otpRepository:  otpRepository,
		userRepository: userRepository,
		emailSender:    emailSender,
		cfg:            cfg,
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

	hashedCode := secure.HashOTP(code.String(), []byte(s.cfg.OTP.Secret))

	if err := s.otpRepository.SaveCode(ctx, user.Email, hashedCode); err != nil {
		return fmt.Errorf("saving hash code: %w", err)
	}

	if err := s.emailSender.SendCode(ctx, user.Email, code.String()); err != nil {
		return fmt.Errorf("sending coding: %w", err)
	}

	return nil
}

func (s *otpService) VerifyCode(ctx context.Context, code string, email string) error {
	foundCode, err := s.otpRepository.GetCodeByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("getting otp code: %w", err)
	}

	if foundCode == "" {
		return domain.NewBadRequestError("otp code not found", ErrOtpCodeNotFound)
	}

	if !secure.VerifyOTP(code, foundCode, []byte(s.cfg.OTP.Secret)) {
		return domain.NewNotFoundError("the inserted code does not match", ErrInvalidOtpCode)
	}

	if err := s.otpRepository.DeleteCode(ctx, email); err != nil {
		return fmt.Errorf("deleting otp code: %w", err)
	}

	return nil
}
