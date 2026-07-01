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
		if apiErr == nil {
			HandleJSON(w, http.StatusInternalServerError, nil)
			slog.Error(
				"unknown domain error code",
				"code",
				domainErr.Code,
				"error",
				domainErr,
			)
			return
		}

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
	switch err.Code {
	case domain.InvalidInput:
		return clienterrors.NewBadRequestError(err.Message, err)
	case domain.Unauthenticated:
		return clienterrors.NewUnauthorizedError(err.Message, err)
	case domain.PermissionDenied:
		return clienterrors.NewForbiddenError(err.Message, err)
	case domain.ResourceNotFound:
		return clienterrors.NewNotFoundError(err.Message, err)
	case domain.AlreadyExists:
		return clienterrors.NewConflictError(err.Message, err)
	case domain.InvalidState:
		return clienterrors.NewUnprocessableEntityError(err.Message, err)
	default:
		return nil
	}
}
