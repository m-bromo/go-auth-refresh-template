package apierrors

import (
	"net/http"

	"github.com/go-playground/validator/v10"
)

type ApiErr struct {
	Message string   `json:"message"`
	Err     string   `json:"error"`
	Path    string   `json:"path"`
	Code    int      `json:"code"`
	Causes  []Causes `json:"causes,omitempty"`
}

type Causes struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (r *ApiErr) Error() string {
	return r.Message
}

func NewValidationError(validationErr validator.ValidationErrors) *ApiErr {
	causes := make([]Causes, len(validationErr))

	for i, err := range validationErr {
		causes[i] = Causes{
			Field:   err.Field(),
			Message: err.Error(),
		}
	}

	return NewBadRequestValidationError("one or more fields are invalid", causes)

}

// 400
func NewBadRequestError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "bad_request",
		Code:    http.StatusBadRequest,
	}
}

// 400
func NewBadRequestValidationError(message string, causes []Causes) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "bad_request",
		Code:    http.StatusBadRequest,
		Causes:  causes,
	}
}

// 401
func NewUnauthorizedError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "unauthorized",
		Code:    http.StatusUnauthorized,
	}
}

// 403
func NewForbiddenError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "forbidden",
		Code:    http.StatusForbidden,
	}
}

// 404
func NewNotFoundError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "not_found",
		Code:    http.StatusNotFound,
	}
}

// 409
func NewConflictError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "conflict",
		Code:    http.StatusConflict,
	}
}

// 422
func NewUnprocessableEntityError(message string, causes []Causes) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "unprocessable_entity",
		Code:    http.StatusUnprocessableEntity,
		Causes:  causes,
	}
}

// 500
func NewInternalServerError(message string) *ApiErr {
	return &ApiErr{
		Message: message,
		Err:     "internal_server_error",
		Code:    http.StatusInternalServerError,
	}
}
