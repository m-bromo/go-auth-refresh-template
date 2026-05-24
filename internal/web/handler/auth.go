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
	authService         service.AuthService
	refreshTokenService service.RefreshTokenService
}

func NewAuthHandler(authService service.AuthService, refreshTokenService service.RefreshTokenService) *AuthHandler {
	return &AuthHandler{
		authService:         authService,
		refreshTokenService: refreshTokenService,
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

	accessToken, refreshToken, err := h.authService.Login(r.Context(), &domain.User{
		Email:    payload.Email,
		Password: payload.Password,
	})
	if err != nil {
		HandleError(w, err)
		return
	}
	cookie.SetCookie(w, refreshToken)

	response := &models.LoginResponse{
		AccessToken: accessToken,
	}

	HandleJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := cookie.GetCookie(r)
	if err != nil {
		HandleError(w, err)
		return
	}

	newAccessToken, newRefreshToken, err := h.refreshTokenService.Refresh(r.Context(), c.Value)
	if err != nil {
		HandleError(w, err)
		return
	}

	cookie.SetCookie(w, newRefreshToken)

	HandleJSON(w, http.StatusOK, newAccessToken)
}
