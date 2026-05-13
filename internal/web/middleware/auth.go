package middleware

import (
	"context"
	"net/http"

	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/cookie"
	"github.com/m-bromo/go-auth-template/internal/web/handler"
)

const UserIDKey = "user_id"

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
		cookie, err := cookie.GetCookie(r)
		if err != nil {
			handler.HandleError(w, err)
			return
		}

		claims, err := s.jwtService.ValidateToken(cookie.Value)
		if err != nil {
			handler.HandleError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
