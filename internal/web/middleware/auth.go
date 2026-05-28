package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/handler"
)

const UserIDKey = "user_id"

var (
	ErrTokenNotProvided = errors.New("the jwt token was not provided")
	ErrUserForbidden    = errors.New("the requested user is not the authenticated subject")
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
		token := r.Header.Get("Authorization")

		claims, err := s.jwtService.ValidateAccessToken(token)
		if err != nil {
			handler.HandleError(w, err)
			return
		}

		if requestedUserID := r.PathValue("id"); requestedUserID != claims.Subject {
			handler.HandleError(w, fmt.Errorf("validating authenticated subject: %w", clienterrors.NewForbiddenError("you are not allowed to access this user", ErrUserForbidden)))
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.Subject)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
