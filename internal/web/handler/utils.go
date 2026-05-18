package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	apierrors "github.com/m-bromo/go-auth-template/internal/api_errors"
)

func HandleJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "Json")

	w.WriteHeader(code)

	json.NewEncoder(w).Encode(body)
}

func HandleError(w http.ResponseWriter, err error) {
	var validationErr validator.ValidationErrors
	var apiErr *apierrors.ClientErr

	if errors.As(err, &validationErr) {
		apiErr = apierrors.NewValidationError(validationErr)
		slog.Warn("failed to validate", "error", apiErr)
		HandleJSON(w, apiErr.Code, apiErr)
		return
	}

	if errors.As(err, &apiErr) {
		slog.Warn("client error", "error", apiErr.Err)
		HandleJSON(w, apiErr.Code, apiErr)
		return
	}

	HandleJSON(w, http.StatusInternalServerError, nil)
	slog.Error("an unexpected internal error has ocurred", "error", err.Error())

}
