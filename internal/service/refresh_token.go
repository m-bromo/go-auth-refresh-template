package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/repository"
)

type RefreshTokenService interface {
	GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error)
	Refresh(ctx context.Context, tokenID string) (string, string, error)
}

type refreshTokenService struct {
	refreshTokenRepository repository.RefreshTokenRepository
	jwtService             JwtService
}

var (
	ErrInvalidRefreshToken           = errors.New("invalid refresh token")
	ErrRefreshTokenNotFoundOrExpired = errors.New("refresh token not found or expired")
)

func NewRefreshTokenService(refreshTokenRepository repository.RefreshTokenRepository, jwtService JwtService) RefreshTokenService {
	return &refreshTokenService{
		refreshTokenRepository: refreshTokenRepository,
		jwtService:             jwtService,
	}
}

func (s *refreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	refreshToken := domain.RefreshToken{
		ID:     uuid.New(),
		UserID: userID,
	}

	if err := s.refreshTokenRepository.Save(ctx, &refreshToken); err != nil {
		return nil, fmt.Errorf("saving refresh token to repository: %w", err)
	}

	return &refreshToken, nil
}

func (s *refreshTokenService) Refresh(ctx context.Context, tokenID string) (string, string, error) {
	tokenIDstring, err := uuid.Parse(tokenID)
	if err != nil {
		return "", "", clienterrors.NewUnauthorizedError("invalid refresh token", ErrInvalidRefreshToken)
	}

	userID, err := s.refreshTokenRepository.Consume(ctx, tokenIDstring)
	if err != nil {
		return "", "", fmt.Errorf("fetching refresh token from repository: %w", err)
	}

	if userID == "" {
		return "", "", clienterrors.NewUnauthorizedError("token not found or expired", ErrRefreshTokenNotFoundOrExpired)
	}

	userIDString, err := uuid.Parse(userID)
	if err != nil {
		return "", "", fmt.Errorf("parsing user id: %w", err)
	}

	newToken := domain.RefreshToken{
		ID:     uuid.New(),
		UserID: userIDString,
	}

	if err := s.refreshTokenRepository.Save(ctx, &newToken); err != nil {
		return "", "", fmt.Errorf("saving new refresh token to repository: %w", err)
	}

	accessToken, err := s.jwtService.GenerateAccessToken(newToken.UserID)
	if err != nil {
		return "", "", fmt.Errorf("generating new access token: %w", err)
	}

	return accessToken, newToken.ID.String(), nil
}
