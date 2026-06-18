package domain

import "errors"

type ErrorType string

const (
	BadRequest          ErrorType = "BAD_REQUEST"
	Unauthorized        ErrorType = "UNAUTHORIZED"
	Forbidden           ErrorType = "FORBIDDEN"
	NotFound            ErrorType = "NOT_FOUND"
	Conflict            ErrorType = "CONFLICT"
	UnprocessableEntity ErrorType = "UNPROCESSABLE_ENTITY"
)

type DomainError struct {
	Err       error
	ErrorType ErrorType
	Message   string
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(message string, errType ErrorType, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: errType,
		Message:   message,
	}
}

func NewBadRequestError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: BadRequest,
		Message:   message,
	}
}

func NewUnauthorizedError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: Unauthorized,
		Message:   message,
	}
}

func NewForbiddenError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: Forbidden,
		Message:   message,
	}
}

func NewNotFoundError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: NotFound,
		Message:   message,
	}
}

func NewConflictError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: Conflict,
		Message:   message,
	}
}

func NewUnprocessableEntityError(message string, errs ...error) *DomainError {
	return &DomainError{
		Err:       errors.Join(errs...),
		ErrorType: UnprocessableEntity,
		Message:   message,
	}
}

func NewConflicError(message string, errs ...error) *DomainError {
	return NewConflictError(message, errs...)
}
