package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m-bromo/go-auth-template/config"
)

func TestNewConfigLoadsDefaultsForOptionalEnvironmentValues(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envPath, []byte("ENVIRONMENT=test\n"), 0o600); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	unsetEnvForTest(
		t,
		"ENVIRONMENT",
		"REDIS_PASSWORD",
		"JWT_PRIVATE_KEY",
		"RESET_TOKEN_SECRET",
		"OTP_SECRET",
		"RESEND_API_KEY",
	)

	cfg, err := config.NewConfig(envPath)
	if err != nil {
		t.Fatalf("NewConfig() error = %v, want nil", err)
	}

	if cfg.Redis.Password != "" {
		t.Errorf("Redis.Password = %q, want empty string", cfg.Redis.Password)
	}

	if cfg.Jwt.PrivateKey != "change-me" {
		t.Errorf("Jwt.PrivateKey = %q, want %q", cfg.Jwt.PrivateKey, "change-me")
	}

	if cfg.ResetToken.Secret != "change-me" {
		t.Errorf("ResetToken.Secret = %q, want %q", cfg.ResetToken.Secret, "change-me")
	}

	if cfg.OTP.Secret != "change-me" {
		t.Errorf("OTP.Secret = %q, want %q", cfg.OTP.Secret, "change-me")
	}

	if cfg.Resend.ApiKey != "" {
		t.Errorf("Resend.ApiKey = %q, want empty string", cfg.Resend.ApiKey)
	}
}

func unsetEnvForTest(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		value, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset %s: %v", key, err)
		}

		t.Cleanup(func() {
			if !exists {
				if err := os.Unsetenv(key); err != nil {
					t.Fatalf("failed to restore unset %s: %v", key, err)
				}
				return
			}

			if err := os.Setenv(key, value); err != nil {
				t.Fatalf("failed to restore %s: %v", key, err)
			}
		})
	}
}
