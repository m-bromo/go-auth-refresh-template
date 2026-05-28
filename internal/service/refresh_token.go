package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	clienterrors "github.com/m-bromo/go-auth-template/internal/api_errors"
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
		return "", "", fmt.Errorf("parsing the token id: %w", clienterrors.NewUnauthorizedError("failed to autheticate user", err))
	}

	userID, err := s.refreshTokenRepository.Consume(ctx, tokenIDstring)
	if err != nil {
		return "", "", fmt.Errorf("fetching refresh token from repository: %w", err)
	}

	if userID == "" {
		return "", "", fmt.Errorf("validating refresh token existence: %w", clienterrors.NewUnauthorizedError("token not found or expired", err))
	}

	newToken := domain.RefreshToken{
		ID:     uuid.New(),
		UserID: uuid.MustParse(userID),
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
