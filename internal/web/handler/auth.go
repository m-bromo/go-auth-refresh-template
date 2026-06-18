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
	otpService          service.OtpService
	cookieManager       *cookie.CookieManager
}

func NewAuthHandler(
	authService service.AuthService,
	refreshTokenService service.RefreshTokenService,
	otpService service.OtpService,
	cookieManager *cookie.CookieManager,
) *AuthHandler {
	return &AuthHandler{
		authService:         authService,
		refreshTokenService: refreshTokenService,
		cookieManager:       cookieManager,
		otpService:          otpService,
	}
}

func (h *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var payload models.RegisterUserPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := validation.Validator.Struct(payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := h.authService.RegisterUser(r.Context(), &domain.User{
		Email:    payload.Email,
		Password: payload.Password,
		Username: payload.Username,
	}); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload models.LoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := validation.Validator.Struct(payload); err != nil {
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
	h.cookieManager.SetCookie(w, refreshToken)

	response := &models.LoginResponse{
		AccessToken: accessToken,
	}

	HandleJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) SendOtpLoginCode(w http.ResponseWriter, r *http.Request) {
	var payload models.SendOtpLoginCodePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := validation.Validator.Struct(payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := h.otpService.SendCode(r.Context(), payload.Email); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) LoginWithOtp(w http.ResponseWriter, r *http.Request) {
	var payload models.LoginWithOtpPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		HandleError(w, err)
		return
	}

	if err := validation.Validator.Struct(payload); err != nil {
		HandleError(w, err)
		return
	}

	accessToken, refreshToken, err := h.authService.LoginWithOtp(r.Context(), payload.Email, payload.Code)
	if err != nil {
		HandleError(w, err)
		return
	}
	h.cookieManager.SetCookie(w, refreshToken)

	response := &models.LoginResponse{
		AccessToken: accessToken,
	}

	HandleJSON(w, http.StatusOK, response)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := h.cookieManager.GetCookie(r)
	if err != nil {
		HandleError(w, domain.NewUnauthorizedError("refresh token was not provided", service.ErrInvalidRefreshToken))
		return
	}

	newAccessToken, newRefreshToken, err := h.refreshTokenService.Refresh(r.Context(), c.Value)
	if err != nil {
		HandleError(w, err)
		return
	}

	h.cookieManager.SetCookie(w, newRefreshToken)

	HandleJSON(w, http.StatusOK, newAccessToken)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := h.cookieManager.GetCookie(r)
	if err != nil {
		h.cookieManager.DeleteCookie(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.refreshTokenService.Revoke(r.Context(), cookie.Value); err != nil {
		h.cookieManager.DeleteCookie(w)
		HandleError(w, err)
		return
	}

	h.cookieManager.DeleteCookie(w)

	w.WriteHeader(http.StatusNoContent)
}
