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

type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	Delete(ctx context.Context, tokenID uuid.UUID) error
}

type refreshTokenService struct {
	refreshTokenOptions *configs.RefreshToken
	unitOfWork          repository.UnitOfWork
	refreshTokenStore   RefreshTokenRepository
	jwtService          JwtService
}

var (
	ErrInvalidRefreshToken           = errors.New("invalid refresh token")
	ErrRefreshTokenNotFoundOrExpired = errors.New("refresh token not found or expired")
)

func NewRefreshTokenService(
	refreshTokenOptions *configs.RefreshToken,
	unitOfWork repository.UnitOfWork,
	refreshTokenStore RefreshTokenRepository,
	jwtService JwtService,
) RefreshTokenService {
	return &refreshTokenService{
		refreshTokenOptions: refreshTokenOptions,
		unitOfWork:          unitOfWork,
		refreshTokenStore:   refreshTokenStore,
		jwtService:          jwtService,
	}
}

func (s *refreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	refreshToken := domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.refreshTokenOptions.Duration),
	}

	if err := s.refreshTokenStore.Save(ctx, &refreshToken); err != nil {
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
			ExpiresAt: time.Now().Add(s.refreshTokenOptions.Duration),
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

	if err := s.refreshTokenStore.Delete(ctx, tokenID); err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}

	return nil
}
