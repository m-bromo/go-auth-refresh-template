package routes

import (
	"github.com/m-bromo/go-auth-template/internal/web/handler"
	"github.com/m-bromo/go-auth-template/internal/web/middleware"
	"github.com/m-bromo/go-auth-template/internal/web/server"
)

type Dependencies struct {
	AuthHandler    *handler.AuthHandler
	UserHandler    *handler.UserHandler
	AuthMiddleware middleware.AuthMiddleware
}

func SetupRoutes(server *server.Server, d Dependencies) {
	server.POST("/auth/register", d.AuthHandler.RegisterUser)
	server.POST("/auth/login", d.AuthHandler.Login)
	server.POST("/auth/logout", d.AuthHandler.Logout)

	server.GET("/user/{id}", d.UserHandler.GetProfile, d.AuthMiddleware.Authenticate)

	server.POST("/refresh", d.AuthHandler.Refresh)
}
