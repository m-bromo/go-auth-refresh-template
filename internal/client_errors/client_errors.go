package clienterrors

import (
	"net/http"

	"github.com/go-playground/validator/v10"
)

type ClientErr struct {
	Err     error    `json:"-"`
	Message string   `json:"message"`
	Code    int      `json:"code"`
	Causes  []Causes `json:"causes,omitempty"`
}

type Causes struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (r *ClientErr) Error() string {
	return r.Message
}

func NewValidationError(validationErr validator.ValidationErrors) *ClientErr {
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
func NewBadRequestError(message string, err error) *ClientErr {
	return &ClientErr{
		Err:     err,
		Message: message,
		Code:    http.StatusBadRequest,
	}
}

// 400
func NewBadRequestValidationError(message string, causes []Causes) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusBadRequest,
		Causes:  causes,
	}
}

// 401
func NewUnauthorizedError(message string, err error) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusUnauthorized,
		Err:     err,
	}
}

// 403
func NewForbiddenError(message string, err error) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusForbidden,
		Err:     err,
	}
}

// 404
func NewNotFoundError(message string, err error) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusNotFound,
		Err:     err,
	}
}

// 409
func NewConflictError(message string, err error) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusConflict,
		Err:     err,
	}
}

// 422
func NewUnprocessableEntityError(message string, err error) *ClientErr {
	return &ClientErr{
		Message: message,
		Code:    http.StatusUnprocessableEntity,
		Err:     err,
	}
}
