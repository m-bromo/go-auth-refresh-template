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
	Refresh(ctx context.Context, tokenID string) (*domain.RefreshToken, error)
}

type refreshTokenService struct {
	refreshTokenRepository repository.RefreshTokenRepository
}

func NewRefreshTokenService(refreshTokenRepository repository.RefreshTokenRepository) RefreshTokenService {
	return &refreshTokenService{
		refreshTokenRepository: refreshTokenRepository,
	}
}

func (s *refreshTokenService) GenerateRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	refreshToken := domain.RefreshToken{
		ID:     uuid.New(),
		UserID: userID,
	}

	if err := s.refreshTokenRepository.Save(ctx, &refreshToken); err != nil {
		return nil, fmt.Errorf("generate token :%w", err)
	}

	return &refreshToken, nil
}

func (s *refreshTokenService) Refresh(ctx context.Context, tokenID string) (*domain.RefreshToken, error) {
	userID, err := s.refreshTokenRepository.Get(ctx, uuid.MustParse(tokenID))
	if err != nil {
		return nil, fmt.Errorf("generate token :%w", err)
	}

	if userID == "" {
		return nil, fmt.Errorf("refresh: %w", clienterrors.NewUnauthorizedError("token not found or expired", err))
	}

	if err := s.refreshTokenRepository.Delete(ctx, uuid.MustParse(tokenID)); err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	newToken := domain.RefreshToken{
		ID:     uuid.New(),
		UserID: uuid.MustParse(userID),
	}

	if err := s.refreshTokenRepository.Save(ctx, &newToken); err != nil {
		return nil, fmt.Errorf("refresh: %w", err)
	}

	return &newToken, nil
}
