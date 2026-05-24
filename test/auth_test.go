package service_test

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"testing"
	"time"

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

	m.Run()
}

func TestRegisterUser_Integration(t *testing.T) {
	ctx := context.Background()

	cfg, err := config.NewConfig("../.env")
	if err != nil {
		log.Fatalf("failed to setup config: %v", err)
	}

	querier := sqlc.New(db)
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService)

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

		if errors.Is(err, service.ErrUserAlreadyRegistered) {
			t.Errorf("expected bad request error for duplicate email, but got: %v", err)
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
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService)

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
