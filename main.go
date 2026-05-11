package main

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/m-bromo/go-auth-template/internal/infra/database"
	"github.com/m-bromo/go-auth-template/internal/infra/database/sqlc"
	"github.com/m-bromo/go-auth-template/internal/repository"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/handler"
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
	userRepository := repository.NewUserRepository(querier)
	authService := service.NewAuthService(userRepository)
	jwtService := service.NewJwtService(cfg)
	authHandler := handler.NewAuthHandler(authService, jwtService)

	srv.POST("/auth/register", authHandler.RegisterUser)
	srv.POST("/auth/login", authHandler.Login)

	if err := srv.Run(fmt.Sprintf("%s:%s", cfg.API.Host, cfg.API.Port)); err != nil {
		log.Fatal(err)
	}
}
