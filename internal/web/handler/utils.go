package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	clienterrors "github.com/m-bromo/go-auth-template/internal/client_errors"
)

func HandleJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func HandleError(w http.ResponseWriter, err error) {
	var validationErr validator.ValidationErrors
	var apiErr *clienterrors.ClientErr

	if err == nil {
		HandleJSON(w, http.StatusInternalServerError, nil)
		slog.Error("an unexpected internal error has occurred")
		return
	}

	if errors.As(err, &validationErr) {
		apiErr = clienterrors.NewValidationError(validationErr)
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
	slog.Error("an unexpected internal error has occurred", "error", err)

}
