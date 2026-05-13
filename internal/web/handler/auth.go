package handler

import (
	"encoding/json"
	"net/http"

	"github.com/m-bromo/go-auth-template/internal/domain"
	"github.com/m-bromo/go-auth-template/internal/pkg/validation"
	"github.com/m-bromo/go-auth-template/internal/service"
	"github.com/m-bromo/go-auth-template/internal/web/cookie"
	"github.com/m-bromo/go-auth-template/internal/web/models"
)

type AuthHandler struct {
	authService service.AuthService
	jwtService  service.JwtService
}

func NewAuthHandler(authService service.AuthService, jwtService service.JwtService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		jwtService:  jwtService,
	}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var paylaod models.RegisterUserPayload
	if err := json.NewDecoder(r.Body).Decode(&paylaod); err != nil {
		HandleError(w, err)
		return
	}

	if err := validation.Validator.Struct(paylaod); err != nil {
		HandleError(w, err)
		return
	}

	if err := h.authService.RegisterUser(r.Context(), &domain.User{
		Email:    paylaod.Email,
		Password: paylaod.Password,
		Username: paylaod.Username,
	}); err != nil {
		HandleError(w, err)
		return
	}

	HandleJSON(w, http.StatusCreated, nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload models.LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, err)
		return
	}

	user, err := h.authService.Login(r.Context(), &domain.User{
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		HandleError(w, err)
		return
	}

	tokenString, err := h.jwtService.GenerateToken(user.ID)
	if err != nil {
		HandleError(w, err)
		return
	}

	cookie.SetCookie(w, tokenString)

	HandleJSON(w, http.StatusOK, nil)
}
