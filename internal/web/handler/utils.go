package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	apierrors "github.com/m-bromo/go-auth-template/internal/web/api_errors"
)

func HandleJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "Json")

	w.WriteHeader(code)

	json.NewEncoder(w).Encode(body)
}

func HandleError(w http.ResponseWriter, err error) {
	var validationErr validator.ValidationErrors
	var apiErr *apierrors.ApiErr

	if errors.As(err, &validationErr) {
		apiErr = apierrors.NewValidationError(err)
	} else {
		switch err {
		default:
			apiErr = apierrors.NewInternalServerError("an unexpected error has ocurred")
		}
	}

	slog.Error(err.Error())
	HandleJSON(w, http.StatusInternalServerError, apiErr)
}
