//go:build integration

package service_test

import (
	"context"
	"database/sql"
	"errors"
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

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/infra/email"
	"github.com/m-bromo/go-auth-template/internal/pkg/secure"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
)

var db *sql.DB
var redisClient *redis.Client

func TestMain(m *testing.M) {
	ctx := context.Background()

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(cfg.Postgres.Name),
		postgres.WithUsername(cfg.Postgres.User),
		postgres.WithPassword(cfg.Postgres.Password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
		testcontainers.WithLogger(log.New(io.Discard, "", 0)),
	)
	if err != nil {
		log.Fatalf("failed to run container: %v", err)
	}

	redisContainer, err := tcredis.Run(ctx,
		"redis:latest",
		tcredis.WithSnapshotting(10, 1),
		tcredis.WithLogLevel(tcredis.LogLevelDebug),
	)
	if err != nil {
		log.Fatalf("failed to run redis container: %v", err)
	}

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate postgres container: %v", err)
		}

		if err := redisContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate redis container: %v", err)
		}
	}()

	rdDsn, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("failed to get redis connection string: %v", err)
	}

	redisOpts, err := redis.ParseURL(rdDsn)
	if err != nil {
		log.Fatalf("failed to parse redis connection string: %v", err)
	}

	redisOpts.Protocol = 2
	redisClient = redis.NewClient(redisOpts)
	defer redisClient.Close()

	pgDsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get postgres connection string: %v", err)
	}

	db, err = sql.Open("postgres", pgDsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	if err := goose.Up(db, "../internal/infra/database/schema"); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	os.Exit(m.Run())
}

func TestRegisterUser_Integration(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	querier := sqlc.New(db)
	emailSender := email.NewEmailSender(cfg)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	otpService := service.NewOtpService(otpRepository, userRepository, emailSender, cfg)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService, otpService)

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

		if domainErr.ErrorType != domain.Conflict {
			t.Errorf("expected duplicate email to return error type %q, got %q", domain.Conflict, domainErr.ErrorType)
		}

		if !errors.Is(domainErr, service.ErrUserAlreadyRegistered) {
			t.Errorf("expected duplicate email cause, got: %v", domainErr.Err)
		}
	})
}

func TestLogin_Integration(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	querier := sqlc.New(db)
	emailSender := email.NewEmailSender(cfg)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	otpService := service.NewOtpService(otpRepository, userRepository, emailSender, cfg)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService, otpService)

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

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	querier := sqlc.New(db)
	emailSender := email.NewEmailSender(cfg)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	otpService := service.NewOtpService(otpRepository, userRepository, emailSender, cfg)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService, otpService)

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

	userID, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch refresh token from redis: %v", err)
	}

	if userID == "" {
		t.Fatalf("expected refresh token to be stored in redis")
	}

	savedCode, err := otpRepository.GetCodeByEmail(ctx, user.Email)
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

	if domainErr.ErrorType != domain.BadRequest {
		t.Errorf("expected consumed otp to return error type %q, got %q", domain.BadRequest, domainErr.ErrorType)
	}

	if !errors.Is(domainErr, service.ErrOtpCodeNotFound) {
		t.Errorf("expected consumed otp cause, got: %v", domainErr.Err)
	}
}

func TestRefreshToken_Integration(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	querier := sqlc.New(db)
	emailSender := email.NewEmailSender(cfg)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	otpService := service.NewOtpService(otpRepository, userRepository, emailSender, cfg)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService, otpService)

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

	userID, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch refresh token from redis: %v", err)
	}

	if userID == "" {
		t.Fatalf("expected refresh token to be stored in redis")
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

	oldTokenUserID, err := refreshTokenRepository.Get(ctx, refreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch old refresh token from redis: %v", err)
	}

	if oldTokenUserID != "" {
		t.Errorf("expected old refresh token to be deleted from redis")
	}

	newRefreshTokenID, err := uuid.Parse(newRefreshToken)
	if err != nil {
		t.Fatalf("expected new refresh token to be a valid UUID, got %q: %v", newRefreshToken, err)
	}

	newTokenUserID, err := refreshTokenRepository.Get(ctx, newRefreshTokenID)
	if err != nil {
		t.Fatalf("failed to fetch new refresh token from redis: %v", err)
	}

	if newTokenUserID != userID {
		t.Errorf("expected new refresh token user ID %q, got %q", userID, newTokenUserID)
	}

	claims, err := jwtService.ValidateAccessToken("Bearer " + newAccessToken)
	if err != nil {
		t.Fatalf("expected new access token to be valid, got: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("expected access token subject %q, got %q", userID, claims.Subject)
	}

	_, _, err = refreshTokenService.Refresh(ctx, refreshToken)
	if err == nil {
		t.Fatalf("expected old refresh token to be rejected after rotation")
	}

	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected old refresh token error to wrap a domain error, got: %v", err)
	}

	if domainErr.ErrorType != domain.Unauthorized {
		t.Errorf("expected old refresh token to return error type %q, got %q", domain.Unauthorized, domainErr.ErrorType)
	}
}
