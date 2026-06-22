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
	server.GET("/swagger", handler.RedirectSwagger)
	server.GET("/swagger/", handler.SwaggerUI)
	server.GET("/swagger/openapi.yaml", handler.SwaggerSpec)

	server.POST("/auth/register", d.AuthHandler.RegisterUser)
	server.POST("/auth/login", d.AuthHandler.Login)
	server.POST("/auth/otp/send", d.AuthHandler.SendOTP)
	server.POST("/auth/login/otp", d.AuthHandler.LoginWithOtp)
	server.POST("/auth/logout", d.AuthHandler.Logout)
	server.POST("/auth/password/verify", d.AuthHandler.VerifyPasswordResetCode)
	server.POST("/auth/password/reset", d.AuthHandler.ResetPassword)

	server.GET("/user/{id}", d.UserHandler.GetProfile, d.AuthMiddleware.Authenticate)

	server.POST("/refresh", d.AuthHandler.Refresh)
}
