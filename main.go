package main

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/infra/cache"
	"github.com/m-bromo/go-auth-template/internal/infra/database"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
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
	userRepository := repository.NewUserRepository(querier)
	refreshTokenRepository := repository.NewRefreshTokenRepository(redisClient, cfg)
	userService := service.NewUserService(userRepository)
	jwtService := service.NewJwtService(cfg)
	refreshTokenService := service.NewRefreshTokenService(refreshTokenRepository, jwtService)
	authService := service.NewAuthService(userRepository, jwtService, refreshTokenService)
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	authHandler := handler.NewAuthHandler(authService, refreshTokenService)
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
