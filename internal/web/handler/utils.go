package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/m-bromo/go-auth-template/internal/domain"
	clienterrors "github.com/m-bromo/go-auth-template/internal/web/client_errors"
)

func HandleJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func HandleHTML(w http.ResponseWriter, code int, body bytes.Buffer) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(code)

	if _, err := w.Write(body.Bytes()); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func HandleError(w http.ResponseWriter, err error) {
	var validationErr validator.ValidationErrors
	var apiErr *clienterrors.ClientErr
	var domainErr *domain.DomainError

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

	if errors.As(err, &domainErr) {
		apiErr = newClientErrorFromDomainError(domainErr)
		slog.Warn("domain error", "error", domainErr)
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

func newClientErrorFromDomainError(err *domain.DomainError) *clienterrors.ClientErr {
	switch err.ErrorType {
	case domain.BadRequest:
		return clienterrors.NewBadRequestError(err.Message, err)
	case domain.Unauthorized:
		return clienterrors.NewUnauthorizedError(err.Message, err)
	case domain.Forbidden:
		return clienterrors.NewForbiddenError(err.Message, err)
	case domain.NotFound:
		return clienterrors.NewNotFoundError(err.Message, err)
	case domain.Conflict:
		return clienterrors.NewConflictError(err.Message, err)
	case domain.UnprocessableEntity:
		return clienterrors.NewUnprocessableEntityError(err.Message, err)
	default:
		return clienterrors.NewUnprocessableEntityError(err.Message, err)
	}
}
