package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	apierrors "github.com/m-bromo/go-auth-template/internal/api_errors"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/handler"
)

const UserIDKey = "user_id"

var (
	ErrTokenNotProvided = errors.New("the jwt token was not provided")
)

type AuthMiddleware interface {
	Authenticate(next http.Handler) http.Handler
}

type authMiddleware struct {
	jwtService service.JwtService
}

func NewAuthMiddleware(jwtService service.JwtService) AuthMiddleware {
	return &authMiddleware{
		jwtService: jwtService,
	}
}

func (s *authMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearerToken := r.Header.Get("Authorization")
		if bearerToken == "" {
			handler.HandleError(w, apierrors.NewUnauthorizedError("failed to get authorization token", ErrTokenNotProvided))
			return
		}

		token := strings.TrimPrefix(bearerToken, "Bearer ")
		claims, err := s.jwtService.ValidateAccessToken(token)
		if err != nil {
			handler.HandleError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
