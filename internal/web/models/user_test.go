package models_test

import (
	"testing"

	"github.com/m-bromo/go-auth-template/internal/pkg/validation"
	"github.com/m-bromo/go-auth-template/internal/web/models"
)

func TestResetPasswordValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload models.ResetPasswordPayload
		wantErr bool
	}{
		{
			name: "accepts strong password and reset token",
			payload: models.ResetPasswordPayload{
				Password:   "new-password@123",
				ResetToken: "reset-token",
			},
		},
		{
			name: "rejects missing password",
			payload: models.ResetPasswordPayload{
				ResetToken: "reset-token",
			},
			wantErr: true,
		},
		{
			name: "rejects weak password",
			payload: models.ResetPasswordPayload{
				Password:   "password",
				ResetToken: "reset-token",
			},
			wantErr: true,
		},
		{
			name: "rejects missing reset token",
			payload: models.ResetPasswordPayload{
				Password: "new-password@123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validation.Validator.Struct(tt.payload)
			if tt.wantErr && err == nil {
				t.Fatalf("Struct() error = nil, want validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("Struct() error = %v, want nil", err)
			}
		})
	}
}
