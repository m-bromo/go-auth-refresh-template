package app

import (
	"database/sql"

	"github.com/m-bromo/go-auth-template/config"
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

func New(cfg *config.Config) (*App, error) {
	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		return nil, err
	}

	redisClient := cache.NewRedisClient(cfg)

	dependencies := setupDependencies(cfg, db, redisClient)

	srv := server.New(cfg)

	routes.SetupRoutes(srv, dependencies)

	return &App{
		Server: srv,
		DB:     db,
		Redis:  redisClient,
	}, nil
}

func setupDependencies(cfg *config.Config, db *sql.DB, redisClient *redis.Client) routes.Dependencies {
	querier := sqlc.New(db)
	emailSender := email.NewEmailSender(cfg)

	userRepository := repository.NewUserRepository(querier)
	resetTokenRepository := repository.NewResetTokenRepository(querier)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
	unitOfWork := repository.NewUnitOfWork(cfg, db, querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(querier, cfg)

	userService := service.NewUserService(userRepository)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(cfg, refreshTokenRepository, jwtService)
	otpService := service.NewOtpService(otpRepository, userRepository, resetTokenRepository, emailSender, cfg)
	authService := service.NewAuthService(
		cfg,
		unitOfWork,
		userRepository,
		resetTokenRepository,
		jwtService,
		refreshTokenService,
		otpService,
	)

	cookieManager := cookie.NewCookieManager(cfg)

	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	authHandler := handler.NewAuthHandler(authService, refreshTokenService, otpService, cookieManager)
	userHandler := handler.NewUserHandler(userService)

	return routes.Dependencies{
		AuthMiddleware: authMiddleware,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
	}
}
