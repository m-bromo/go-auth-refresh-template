package domain

import "errors"

type ErrorCode string

const (
	InvalidInput     ErrorCode = "INVALID_INPUT"
	Unauthenticated  ErrorCode = "UNAUTHENTICATED"
	PermissionDenied ErrorCode = "PERMISSION_DENIED"
	ResourceNotFound ErrorCode = "RESOURCE_NOT_FOUND"
	AlreadyExists    ErrorCode = "ALREADY_EXISTS"
	InvalidState     ErrorCode = "INVALID_STATE"
)

type DomainError struct {
	Err     error
	Code    ErrorCode
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func NewDomainError(message string, code ErrorCode, errs ...error) *DomainError {
	return &DomainError{
		Err:     errors.Join(errs...),
		Code:    code,
		Message: message,
	}
}

func NewInvalidInputError(message string, errs ...error) *DomainError {
	return NewDomainError(message, InvalidInput, errs...)
}

func NewUnauthenticatedError(message string, errs ...error) *DomainError {
	return NewDomainError(message, Unauthenticated, errs...)
}

func NewPermissionDeniedError(message string, errs ...error) *DomainError {
	return NewDomainError(message, PermissionDenied, errs...)
}

func NewResourceNotFoundError(message string, errs ...error) *DomainError {
	return NewDomainError(message, ResourceNotFound, errs...)
}

func NewAlreadyExistsError(message string, errs ...error) *DomainError {
	return NewDomainError(message, AlreadyExists, errs...)
}

func NewInvalidStateError(message string, errs ...error) *DomainError {
	return NewDomainError(message, InvalidState, errs...)
}
