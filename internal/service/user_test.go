package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/mocks"
	"github.com/m-bromo/go-auth-template/internal/service"
)

func TestUserService_GetProfile(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repositoryErr := errors.New("repository failed")
	expectedUser := &domain.User{
		ID:       userID,
		Email:    "user@test.com",
		Username: "user",
	}

	tests := []struct {
		name          string
		id            string
		user          *domain.User
		getByIDErr    error
		wantUser      *domain.User
		wantErr       error
		wantErrType   domain.ErrorType
		wantWrapped   string
		wantRepoCalls int
	}{
		{
			name:          "returns user profile",
			id:            userID.String(),
			user:          expectedUser,
			wantUser:      expectedUser,
			wantRepoCalls: 1,
		},
		{
			name:        "rejects invalid user id",
			id:          "not-a-uuid",
			wantErr:     service.ErrInvalidUserID,
			wantErrType: domain.BadRequest,
		},
		{
			name:          "wraps repository error",
			id:            userID.String(),
			getByIDErr:    repositoryErr,
			wantWrapped:   "fetching user from repository by ID",
			wantRepoCalls: 1,
		},
		{
			name:          "rejects missing user",
			id:            userID.String(),
			wantErr:       service.ErrUserNotFound,
			wantErrType:   domain.NotFound,
			wantRepoCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			userRepository := &mocks.UserRepository{
				GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
					return tt.user, tt.getByIDErr
				},
			}
			userService := service.NewUserService(userRepository)

			user, err := userService.GetProfile(t.Context(), tt.id)

			if tt.wantErr != nil {
				assertDomainError(t, err, tt.wantErrType, tt.wantErr)
			} else if tt.wantWrapped != "" {
				assertWrappedError(t, err, tt.wantWrapped, repositoryErr)
			} else if err != nil {
				t.Fatalf("GetProfile() error = %v, want nil", err)
			}

			if user != tt.wantUser {
				t.Errorf("user = %v, want %v", user, tt.wantUser)
			}

			if userRepository.GetByIDCalls != tt.wantRepoCalls {
				t.Errorf("GetByID() calls = %d, want %d", userRepository.GetByIDCalls, tt.wantRepoCalls)
			}
		})
	}
}
