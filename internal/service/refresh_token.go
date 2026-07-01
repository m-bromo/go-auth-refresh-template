package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

type RefreshTokenService interface {
	GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error)
	Refresh(ctx context.Context, tokenIDString string) (string, string, error)
	Revoke(ctx context.Context, tokenIDString string) error
}

type refreshTokenService struct {
	cfg                    *configs.Config
	unitOfWork             repository.UnitOfWork
	refreshTokenRepository repository.RefreshTokenRepository
	jwtService             JwtService
}

var (
	ErrInvalidRefreshToken           = errors.New("invalid refresh token")
	ErrRefreshTokenNotFoundOrExpired = errors.New("refresh token not found or expired")
)

func NewRefreshTokenService(
	cfg *configs.Config,
	unitOfWork repository.UnitOfWork,
	refreshTokenRepository repository.RefreshTokenRepository,
	jwtService JwtService,
) RefreshTokenService {
	return &refreshTokenService{
		cfg:                    cfg,
		unitOfWork:             unitOfWork,
		refreshTokenRepository: refreshTokenRepository,
		jwtService:             jwtService,
	}
}

func (s *refreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	refreshToken := domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.cfg.RefreshToken.Duration),
	}

	if err := s.refreshTokenRepository.Save(ctx, &refreshToken); err != nil {
		return nil, fmt.Errorf("saving refresh token to repository: %w", err)
	}

	return &refreshToken, nil
}

func (s *refreshTokenService) Refresh(ctx context.Context, tokenIDString string) (string, string, error) {
	tokenID, err := uuid.Parse(tokenIDString)
	if err != nil {
		return "", "", domain.NewUnauthorizedError("invalid refresh token", ErrInvalidRefreshToken)
	}

	var accessTokenString, refreshTokenString string
	if err := s.unitOfWork.Exec(ctx, func(repos repository.Repositories) error {
		userID, err := repos.RefreshTokenRepository.Consume(ctx, tokenID)
		if err != nil {
			return fmt.Errorf("fetching refresh token from repository: %w", err)
		}

		if userID == "" {
			return domain.NewUnauthorizedError("token not found or expired", ErrRefreshTokenNotFoundOrExpired)
		}

		userIDString, err := uuid.Parse(userID)
		if err != nil {
			return fmt.Errorf("parsing user id: %w", err)
		}

		newToken := domain.RefreshToken{
			ID:        uuid.New(),
			UserID:    userIDString,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(s.cfg.RefreshToken.Duration),
		}
		refreshTokenString = newToken.ID.String()

		if err := repos.RefreshTokenRepository.Save(ctx, &newToken); err != nil {
			return fmt.Errorf("saving new refresh token to repository: %w", err)
		}

		accessTokenString, err = s.jwtService.GenerateAccessToken(newToken.UserID)
		if err != nil {
			return fmt.Errorf("generating new access token: %w", err)
		}

		return nil
	}); err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *refreshTokenService) Revoke(ctx context.Context, tokenIDString string) error {
	tokenID, err := uuid.Parse(tokenIDString)
	if err != nil {
		return domain.NewUnauthorizedError("invalid refresh token", ErrInvalidRefreshToken)
	}

	if err := s.refreshTokenRepository.Delete(ctx, tokenID); err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}

	return nil
}
