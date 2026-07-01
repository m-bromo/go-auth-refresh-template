package app

import (
	"database/sql"
	"fmt"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/m-bromo/go-auth-template/internal/infra/cache"
	"github.com/m-bromo/go-auth-template/internal/infra/database"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/infra/email"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/cookie"
	"github.com/m-bromo/go-auth-template/internal/web/handler"
	"github.com/m-bromo/go-auth-template/internal/web/middleware"
	"github.com/m-bromo/go-auth-template/internal/web/routes"
	"github.com/m-bromo/go-auth-template/internal/web/server"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Server *server.Server
	DB     *sql.DB
	Redis  *redis.Client
}

func New(configOptions *configs.Config) (*App, error) {
	db, err := database.NewPostgresConnection(&configOptions.Postgres)
	if err != nil {
		return nil, fmt.Errorf("starting postgres database: %w", err)
	}

	redisClient, err := cache.NewRedisClient(&configOptions.Redis)
	if err != nil {
		return nil, fmt.Errorf("stating redis database: %w", err)
	}

	dependencies := setupDependencies(configOptions, db, redisClient)

	srv := server.New(&configOptions.API)

	routes.SetupRoutes(srv, dependencies)

	return &App{
		Server: srv,
		DB:     db,
		Redis:  redisClient,
	}, nil
}

func (a *App) Close() {
	a.DB.Close()
	a.Redis.Close()
}

func setupDependencies(
	configOptions *configs.Config,
	db *sql.DB,
	redisClient *redis.Client,
) routes.Dependencies {
	queries := sqlc.New(db)
	resendClient := email.NewResendClient(&configOptions.Resend)

	sqlcUserRepository := repository.NewSqlcUserRepository(queries)
	sqlcResetTokenRepository := repository.NewSqlcResetTokenRepository(queries)
	redisOtpRepository := repository.NewRedisOtpRepository(redisClient, &configOptions.OTP)
	sqlcRefreshTokenRepository := repository.NewSqlcRefreshTokenRepository(queries)
	unitOfWork := repository.NewUnitOfWork(db, queries)

	userService := service.NewUserService(sqlcUserRepository)
	jwtService := service.NewJwtService(&configOptions.Jwt)
	refreshTokenService := service.NewRefreshTokenService(
		&configOptions.RefreshToken,
		unitOfWork,
		sqlcRefreshTokenRepository,
		jwtService,
	)
	otpService := service.NewOtpService(
		redisOtpRepository,
		sqlcUserRepository,
		sqlcResetTokenRepository,
		resendClient,
		&configOptions.OTP,
		&configOptions.ResetToken,
	)
	authService := service.NewAuthService(
		&configOptions.ResetToken,
		unitOfWork,
		sqlcUserRepository,
		jwtService,
		refreshTokenService,
		otpService,
	)

	cookieManager := cookie.NewCookieManager(
		configOptions.Environment,
		&configOptions.RefreshToken,
	)

	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	authHandler := handler.NewAuthHandler(authService, refreshTokenService, otpService, cookieManager)
	userHandler := handler.NewUserHandler(userService)

	return routes.Dependencies{
		AuthMiddleware: authMiddleware,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
	}
}
