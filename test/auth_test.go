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
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

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
	redisClient       *redis.Client
	postgresContainer *postgres.PostgresContainer
	redisContainer    *tcredis.RedisContainer
}

type authFixture struct {
	config                 *configs.Config
	userRepository         *repository.SqlcUserRepository
	otpRepository          *repository.RedisOtpRepository
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

	redisContainer, err := tcredis.Run(ctx,
		"redis:7-alpine",
		tcredis.WithSnapshotting(10, 1),
		tcredis.WithLogLevel(tcredis.LogLevelDebug),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("6379/tcp").WithStartupTimeout(time.Minute),
			wait.ForLog("* Ready to accept connections").WithStartupTimeout(time.Minute),
		),
		testcontainers.WithLogger(log.New(io.Discard, "", 0)),
	)
	env.redisContainer = redisContainer
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("starting redis container: %w", err),
			env.Close(),
		)
	}

	rdDsn, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("getting redis connection string: %w", err),
			env.Close(),
		)
	}

	redisOpts, err := redis.ParseURL(rdDsn)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("parsing redis connection string: %w", err),
			env.Close(),
		)
	}

	redisOpts.Protocol = 2
	env.redisClient = redis.NewClient(redisOpts)
	if err := env.redisClient.Ping(ctx).Err(); err != nil {
		return nil, errors.Join(
			fmt.Errorf("pinging redis: %w", err),
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
	if e.redisClient != nil {
		if err := e.redisClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing redis client: %w", err))
		}
	}
	if e.redisContainer != nil {
		if err := e.redisContainer.Terminate(ctx); err != nil {
			errs = append(errs, fmt.Errorf("terminating redis container: %w", err))
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
	otpRepository := repository.NewRedisOtpRepository(testEnv.redisClient, &cfg.OTP)
	userRepository := repository.NewSqlcUserRepository(querier)
	resetTokenRepository := repository.NewSqlcResetTokenRepository(querier)
	unitOfWork := repository.NewUnitOfWork(testEnv.db, querier)
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
	if err := otpRepository.SaveCode(ctx, user.Email, hashedCode); err != nil {
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

	savedCode, err := testEnv.redisClient.Get(ctx, user.Email).Result()
	if err == redis.Nil {
		savedCode = ""
		err = nil
	}
	if err != nil {
		t.Fatalf("failed to fetch otp code from redis: %v", err)
	}

	if savedCode != "" {
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
