package main

import (
	"fmt"
	"log"
	"log/slog"

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
)

func main() {
	slog.Info("starting application")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	srv := server.New()
	querier := sqlc.New(db)
	redisClient := cache.NewRedisClient(cfg)
	emailSender := email.NewEmailSender(cfg)
	userRepository := repository.NewUserRepository(querier)
	resetTokenRepository := repository.NewResetTokenRepository(querier)
	unitOfWork := repository.NewUnitOfWork(cfg, db, querier)
	otpRepository := repository.NewOtpRepository(redisClient, cfg)
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

	routes.SetupRoutes(srv, routes.Dependencies{
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		AuthMiddleware: authMiddleware,
	})

	if err := srv.Run(fmt.Sprintf("%s:%s", cfg.API.Host, cfg.API.Port)); err != nil {
		log.Fatal(err)
	}
}
