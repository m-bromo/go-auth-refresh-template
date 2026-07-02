//go:build integration

package service_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/infra/email"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/pkg/secure"
)

const (
	startupTimeout = 2 * time.Minute
	cleanupTimeout = 30 * time.Second
)

var testEnv *integrationEnvironment

type integrationEnvironment struct {
	config            *configs.Config
	db                *sql.DB
	postgresContainer *postgres.PostgresContainer
}

type authFixture struct {
	config                 *configs.Config
	userRepository         *repository.SqlcUserRepository
	otpRepository          *repository.SqlcOtpRepository
	refreshTokenRepository *repository.SqlcRefreshTokenRepository
	authService            service.AuthService
	refreshTokenService    service.RefreshTokenService
	jwtService             service.JwtService
}

func TestMain(m *testing.M) {
	env, err := setupIntegrationEnvironment()
	if err != nil {
		log.Printf("failed to set up integration environment: %v", err)
		os.Exit(1)
	}
	testEnv = env

	code := m.Run()

	if err := env.Close(); err != nil {
		log.Printf("failed to clean up integration environment: %v", err)
		if code == 0 {
			code = 1
		}
	}

	os.Exit(code)
}

func setupIntegrationEnvironment() (*integrationEnvironment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	cfg, err := configs.NewConfig("../.env")
	if err != nil {
		return nil, fmt.Errorf("loading test configuration: %w", err)
	}

	env := &integrationEnvironment{
		config: cfg,
	}

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(cfg.Postgres.Name),
		postgres.WithUsername(cfg.Postgres.User),
		postgres.WithPassword(cfg.Postgres.Password),
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(log.New(io.Discard, "", 0)),
	)
	env.postgresContainer = pgContainer
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("starting postgres container: %w", err),
			env.Close(),
		)
	}

	pgDsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("getting postgres connection string: %w", err),
			env.Close(),
		)
	}

	env.db, err = sql.Open("postgres", pgDsn)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("opening postgres connection: %w", err),
			env.Close(),
		)
	}
	if err := env.db.PingContext(ctx); err != nil {
		return nil, errors.Join(
			fmt.Errorf("pinging postgres: %w", err),
			env.Close(),
		)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return nil, errors.Join(
			fmt.Errorf("setting goose dialect: %w", err),
			env.Close(),
		)
	}

	if err := goose.Up(env.db, "../internal/infra/database/schema"); err != nil {
		return nil, errors.Join(
			fmt.Errorf("running database migrations: %w", err),
			env.Close(),
		)
	}

	return env, nil
}

func (e *integrationEnvironment) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()

	var errs []error
	if e.db != nil {
		if err := e.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing postgres connection: %w", err))
		}
	}
	if e.postgresContainer != nil {
		if err := e.postgresContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("terminating postgres container: %w", err))
		}
	}

	return errors.Join(errs...)
}

func newAuthFixture() *authFixture {
	cfg := testEnv.config
	querier := sqlc.New(testEnv.db)
	emailSender := email.NewResendClient(&cfg.Resend)
	otpRepository := repository.NewSqlcOtpRepository(querier, &cfg.OTP)
	userRepository := repository.NewSqlcUserRepository(querier)
	resetTokenRepository := repository.NewSqlcResetTokenRepository(querier)
	unitOfWork := repository.NewUnitOfWork(testEnv.db, querier, &cfg.OTP)
	refreshTokenRepository := repository.NewSqlcRefreshTokenRepository(querier)
	jwtService := service.NewJwtService(&cfg.Jwt)
	refreshTokenService := service.NewRefreshTokenService(
		&cfg.RefreshToken,
		unitOfWork,
		refreshTokenRepository,
		jwtService,
	)
	otpService := service.NewOtpService(
		otpRepository,
		unitOfWork,
		userRepository,
		resetTokenRepository,
		emailSender,
		&cfg.OTP,
		&cfg.ResetToken,
	)
	authService := service.NewAuthService(
		&cfg.ResetToken,
		unitOfWork,
		userRepository,
		jwtService,
		refreshTokenService,
		otpService,
	)

	return &authFixture{
		config:                 cfg,
		userRepository:         userRepository,
		otpRepository:          otpRepository,
		refreshTokenRepository: refreshTokenRepository,
		authService:            authService,
		refreshTokenService:    refreshTokenService,
		jwtService:             jwtService,
	}
}

func TestRegisterUser_Integration(t *testing.T) {
	ctx := context.Background()

	fixture := newAuthFixture()
	authService := fixture.authService
	userRepository := fixture.userRepository

	t.Run("should register a user successfully", func(t *testing.T) {
		user := &domain.User{
			Email:    "example@test.com",
			Username: "newUser",
			Password: "password@123",
		}

		if err := authService.RegisterUser(ctx, user); err != nil {
			t.Errorf("did not expect an error when registering user, but got: %v", err)
		}

		savedUser, err := userRepository.GetByEmail(ctx, user.Email)
		if err != nil {
			t.Errorf("failed to fetch saved user: %v", err)
		}

		if savedUser == nil {
			t.Errorf("user was not found in the database")
		}

		if savedUser.Username != user.Username {
			t.Errorf("expected username %s, got %s", user.Username, savedUser.Username)
		}

		if savedUser.Email != user.Email {
			t.Errorf("expected email %s, got %s", user.Email, savedUser.Email)
		}
	})

	t.Run("should fail to register a user with an already existing email", func(t *testing.T) {
		duplicateUser := &domain.User{
			Email:    "example@test.com",
			Username: "newUser",
			Password: "password@123",
		}

		err := authService.RegisterUser(ctx, duplicateUser)
		if err == nil {
			t.Errorf("expected duplicate email error, but got success")
		}

		var domainErr *domain.DomainError
		if !errors.As(err, &domainErr) {
			t.Fatalf("expected duplicate email to wrap a domain error, got: %v", err)
		}

		if domainErr.Code != domain.AlreadyExists {
			t.Errorf("expected duplicate email to return error code %q, got %q", domain.AlreadyExists, domainErr.Code)
		}

		if !errors.Is(domainErr, service.ErrUserAlreadyRegistered) {
			t.Errorf("expected duplicate email cause, got: %v", domainErr.Err)
		}
	})
}

func TestLogin_Integration(t *testing.T) {
	ctx := context.Background()

	fixture := newAuthFixture()
	authService := fixture.authService

	user := &domain.User{
		Email:    "login@test.com",
		Username: "loginUser",
		Password: "password@123",
	}

	if err := authService.RegisterUser(ctx, user); err != nil {
		t.Fatalf("failed to setup user for login test: %v", err)
	}

	t.Run("should login successfully", func(t *testing.T) {
		loginUser := &domain.User{
			Email:    "login@test.com",
			Password: "password@123",
		}

		accessToken, refreshToken, err := authService.Login(ctx, loginUser)

		if err != nil {
			t.Errorf("did not expect an error when logging in, but got: %v", err)
		}

		if accessToken == "" {
			t.Errorf("expected an access token, but got empty string")
		}

		if refreshToken == "" {
			t.Errorf("expected a refresh token, but got empty string")
		}
	})

	t.Run("should fail to login with incorrect password", func(t *testing.T) {
		loginUser := &domain.User{
			Email:    "login@test.com",
			Password: "wrongpassword",
		}

		_, _, err := authService.Login(ctx, loginUser)
		if err == nil {
			t.Errorf("expected error for incorrect password, but got success")
		}
	})

	t.Run("should fail to login with non-existent email", func(t *testing.T) {
		loginUser := &domain.User{
			Email:    "nonexistent@test.com",
			Password: "password@123",
		}

		_, _, err := authService.Login(ctx, loginUser)
		if err == nil {
			t.Errorf("expected error for non-existent email, but got success")
		}
	})
}

func TestLoginWithOtp_Integration(t *testing.T) {
	ctx := context.Background()

	fixture := newAuthFixture()
	cfg := fixture.config
	authService := fixture.authService
	otpRepository := fixture.otpRepository
	refreshTokenRepository := fixture.refreshTokenRepository

	user := &domain.User{
		Email:    "otp-login@test.com",
		Username: "otpLoginUser",
		Password: "password@123",
	}

	if err := authService.RegisterUser(ctx, user); err != nil {
		t.Fatalf("failed to setup user for otp login test: %v", err)
	}

	code := "123456"
	hashedCode := secure.HashOTP(code, []byte(cfg.OTP.Secret))
	otp := &domain.OTP{
		ID:         uuid.New(),
		Identifier: user.Email,
		Code:       hashedCode,
		ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
		CreatedAt:  time.Now(),
	}
	if err := otpRepository.Save(ctx, otp); err != nil {
		t.Fatalf("failed to setup otp code for login test: %v", err)
	}

	accessToken, refreshToken, err := authService.LoginWithOtp(ctx, user.Email, code)
	if err != nil {
		t.Fatalf("did not expect an error when logging in with otp, but got: %v", err)
	}

	if accessToken == "" {
		t.Errorf("expected an access token, but got empty string")
	}

	if refreshToken == "" {
		t.Fatalf("expected a refresh token, but got empty string")
	}

	refreshTokenID, err := uuid.Parse(refreshToken)
	if err != nil {
		t.Fatalf("expected refresh token to be a valid UUID, got %q: %v", refreshToken, err)
	}

	storedRefreshToken, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch refresh token from postgres: %v", err)
	}

	if storedRefreshToken == nil {
		t.Fatalf("expected refresh token to be stored in postgres")
	}

	var otpExists bool
	if err := testEnv.db.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM otp WHERE id = $1)",
		otp.ID,
	).Scan(&otpExists); err != nil {
		t.Fatalf("failed to check consumed otp code in postgres: %v", err)
	}

	if otpExists {
		t.Errorf("expected otp code to be deleted after successful login")
	}

	_, _, err = authService.LoginWithOtp(ctx, user.Email, code)
	if err == nil {
		t.Fatalf("expected consumed otp code to be rejected")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected consumed otp error to wrap a domain error, got: %v", err)
	}

	if domainErr.Code != domain.ResourceNotFound {
		t.Errorf("expected consumed otp to return error code %q, got %q", domain.ResourceNotFound, domainErr.Code)
	}

	if !errors.Is(domainErr, service.ErrInvalidOtpCode) {
		t.Errorf("expected consumed otp cause, got: %v", domainErr.Err)
	}
}

func TestOtpRepository_MaxAttempts_Integration(t *testing.T) {
	ctx := context.Background()
	fixture := newAuthFixture()
	cfg := fixture.config
	otpRepository := fixture.otpRepository

	if cfg.OTP.MaxAttempts <= 0 {
		t.Fatalf("OTP max attempts = %d, want a positive value", cfg.OTP.MaxAttempts)
	}

	t.Run("caps attempts at configured maximum", func(t *testing.T) {
		otp := &domain.OTP{
			ID:         uuid.New(),
			Identifier: "otp-attempt-cap@test.com",
			Code:       secure.HashOTP("123456", []byte(cfg.OTP.Secret)),
			Attempts:   cfg.OTP.MaxAttempts - 1,
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := otpRepository.Save(ctx, otp); err != nil {
			t.Fatalf("failed to save otp: %v", err)
		}

		if err := otpRepository.IncreaseAttempts(ctx, otp.ID); err != nil {
			t.Fatalf("first IncreaseAttempts() error = %v", err)
		}
		if err := otpRepository.IncreaseAttempts(ctx, otp.ID); err != nil {
			t.Fatalf("second IncreaseAttempts() error = %v", err)
		}

		var attempts int
		if err := testEnv.db.QueryRowContext(
			ctx,
			"SELECT attempts FROM otp WHERE id = $1",
			otp.ID,
		).Scan(&attempts); err != nil {
			t.Fatalf("failed to fetch otp attempts: %v", err)
		}

		if attempts != cfg.OTP.MaxAttempts {
			t.Errorf("OTP attempts = %d, want %d", attempts, cfg.OTP.MaxAttempts)
		}
	})

	t.Run("rejects correct code after maximum attempts", func(t *testing.T) {
		const code = "654321"
		otp := &domain.OTP{
			ID:         uuid.New(),
			Identifier: "otp-blocked@test.com",
			Code:       secure.HashOTP(code, []byte(cfg.OTP.Secret)),
			Attempts:   cfg.OTP.MaxAttempts,
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := otpRepository.Save(ctx, otp); err != nil {
			t.Fatalf("failed to save otp: %v", err)
		}

		consumed, err := otpRepository.ConsumeByChallengeID(ctx, otp.ID, otp.Code)
		if err != nil {
			t.Fatalf("ConsumeByChallengeID() error = %v", err)
		}

		if consumed != nil {
			t.Errorf("ConsumeByChallengeID() OTP = %#v, want nil for blocked challenge", consumed)
		}
	})
}

func TestOtpReplacement_Integration(t *testing.T) {
	ctx := context.Background()
	cfg := testEnv.config
	querier := sqlc.New(testEnv.db)
	otpRepository := repository.NewSqlcOtpRepository(querier, &cfg.OTP)
	unitOfWork := repository.NewUnitOfWork(testEnv.db, querier, &cfg.OTP)

	t.Run("replaces previous otp for identifier", func(t *testing.T) {
		identifier := fmt.Sprintf("otp-replace-%s@test.com", uuid.New())
		previousOTP := &domain.OTP{
			ID:         uuid.New(),
			Identifier: identifier,
			Code:       secure.HashOTP("123456", []byte(cfg.OTP.Secret)),
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := otpRepository.Save(ctx, previousOTP); err != nil {
			t.Fatalf("failed to save previous otp: %v", err)
		}

		nextOTP := &domain.OTP{
			ID:         uuid.New(),
			Identifier: identifier,
			Code:       secure.HashOTP("654321", []byte(cfg.OTP.Secret)),
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := unitOfWork.Exec(ctx, func(repos repository.Repositories) error {
			if err := repos.OTPRepository.InvalidateByIdentifier(ctx, identifier); err != nil {
				return err
			}

			return repos.OTPRepository.Save(ctx, nextOTP)
		}); err != nil {
			t.Fatalf("replacing otp transaction error = %v", err)
		}

		var count int
		if err := testEnv.db.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM otp WHERE identifier = $1 AND id = $2",
			identifier,
			nextOTP.ID,
		).Scan(&count); err != nil {
			t.Fatalf("failed to query replaced otp: %v", err)
		}

		if count != 1 {
			t.Errorf("replaced OTP count = %d, want 1", count)
		}
	})

	t.Run("rolls back invalidation when save fails", func(t *testing.T) {
		identifier := fmt.Sprintf("otp-rollback-%s@test.com", uuid.New())
		previousOTP := &domain.OTP{
			ID:         uuid.New(),
			Identifier: identifier,
			Code:       secure.HashOTP("123456", []byte(cfg.OTP.Secret)),
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := otpRepository.Save(ctx, previousOTP); err != nil {
			t.Fatalf("failed to save previous otp: %v", err)
		}

		conflictingOTP := &domain.OTP{
			ID:         uuid.New(),
			Identifier: fmt.Sprintf("otp-conflict-%s@test.com", uuid.New()),
			Code:       secure.HashOTP("000000", []byte(cfg.OTP.Secret)),
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		if err := otpRepository.Save(ctx, conflictingOTP); err != nil {
			t.Fatalf("failed to save conflicting otp: %v", err)
		}

		nextOTP := &domain.OTP{
			ID:         conflictingOTP.ID,
			Identifier: identifier,
			Code:       secure.HashOTP("654321", []byte(cfg.OTP.Secret)),
			ExpiresAt:  time.Now().Add(cfg.OTP.Duration),
			CreatedAt:  time.Now(),
		}
		err := unitOfWork.Exec(ctx, func(repos repository.Repositories) error {
			if err := repos.OTPRepository.InvalidateByIdentifier(ctx, identifier); err != nil {
				return err
			}

			return repos.OTPRepository.Save(ctx, nextOTP)
		})
		if err == nil {
			t.Fatal("replacing otp transaction error = nil, want duplicate key error")
		}

		var previousExists bool
		if err := testEnv.db.QueryRowContext(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM otp WHERE id = $1)",
			previousOTP.ID,
		).Scan(&previousExists); err != nil {
			t.Fatalf("failed to query previous otp after rollback: %v", err)
		}

		if !previousExists {
			t.Error("previous OTP was deleted despite transaction rollback")
		}
	})
}

func TestRefreshToken_Integration(t *testing.T) {
	ctx := context.Background()

	fixture := newAuthFixture()
	authService := fixture.authService
	refreshTokenRepository := fixture.refreshTokenRepository
	refreshTokenService := fixture.refreshTokenService
	jwtService := fixture.jwtService

	password := "password@123"
	user := &domain.User{
		Email:    "refresh@test.com",
		Username: "refreshUser",
		Password: password,
	}

	if err := authService.RegisterUser(ctx, user); err != nil {
		t.Fatalf("failed to setup user for refresh token test: %v", err)
	}

	accessToken, refreshToken, err := authService.Login(ctx, &domain.User{
		Email:    user.Email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("failed to login user for refresh token test: %v", err)
	}

	if accessToken == "" {
		t.Fatalf("expected login to return an access token")
	}

	refreshTokenID, err := uuid.Parse(refreshToken)
	if err != nil {
		t.Fatalf("expected refresh token to be a valid UUID, got %q: %v", refreshToken, err)
	}

	storedRefreshToken, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch refresh token from postgres: %v", err)
	}

	if storedRefreshToken == nil {
		t.Fatalf("expected refresh token to be stored in postgres")
	}

	newAccessToken, newRefreshToken, err := refreshTokenService.Refresh(ctx, refreshToken)
	if err != nil {
		t.Fatalf("did not expect an error when refreshing token, but got: %v", err)
	}

	if newAccessToken == "" {
		t.Errorf("expected a new access token, but got empty string")
	}

	if newAccessToken == accessToken {
		t.Errorf("expected a rotated access token, but got the original token")
	}

	if newRefreshToken == "" {
		t.Fatalf("expected a new refresh token, but got empty string")
	}

	if newRefreshToken == refreshToken {
		t.Errorf("expected refresh token to be rotated")
	}

	oldRefreshToken, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch old refresh token from postgres: %v", err)
	}

	if oldRefreshToken != nil && oldRefreshToken.ExpiresAt.After(time.Now()) {
		t.Errorf("expected old refresh token to be expired")
	}

	newRefreshTokenID, err := uuid.Parse(newRefreshToken)
	if err != nil {
		t.Fatalf("expected new refresh token to be a valid UUID, got %q: %v", newRefreshToken, err)
	}

	newStoredRefreshToken, err := refreshTokenRepository.Get(ctx, newRefreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch new refresh token from postgres: %v", err)
	}

	if newStoredRefreshToken == nil {
		t.Fatalf("expected new refresh token to be stored in postgres")
	}

	if newStoredRefreshToken.UserID != user.ID {
		t.Errorf("expected new refresh token user ID %q, got %q", user.ID, newStoredRefreshToken.UserID)
	}

	claims, err := jwtService.ValidateAccessToken("Bearer " + newAccessToken)
	if err != nil {
		t.Fatalf("expected new access token to be valid, got: %v", err)
	}

	if claims.Subject != user.ID.String() {
		t.Errorf("expected access token subject %q, got %q", user.ID, claims.Subject)
	}

	_, _, err = refreshTokenService.Refresh(ctx, refreshToken)
	if err == nil {
		t.Fatalf("expected old refresh token to be rejected after rotation")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected old refresh token error to wrap a domain error, got: %v", err)
	}

	if domainErr.Code != domain.Unauthenticated {
		t.Errorf("expected old refresh token to return error code %q, got %q", domain.Unauthenticated, domainErr.Code)
	}
}
