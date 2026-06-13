// Package errors defines standard error types for the LightAI Go application.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors.
var (
	ErrNotFound       = NewAppError(http.StatusNotFound, "resource not found")
	ErrConflict       = NewAppError(http.StatusConflict, "resource conflict")
	ErrValidation     = NewAppError(http.StatusBadRequest, "validation error")
	ErrUnauthorized   = NewAppError(http.StatusUnauthorized, "unauthorized")
	ErrForbidden      = NewAppError(http.StatusForbidden, "forbidden")
	ErrInternalServer = NewAppError(http.StatusInternalServerError, "internal server error")
	ErrRateLimited    = NewAppError(http.StatusTooManyRequests, "rate limited")
)

// AppError is an application-level error with an HTTP status code.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

// NewAppError creates a new AppError.
func NewAppError(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is comparison by code and message.
// This allows sentinel errors like ErrNotFound to match even after Wrap.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Message == t.Message
}

// Wrap wraps an existing error with an AppError.
func (e *AppError) Wrap(err error) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Err:     err,
	}
}

// IsAppError checks if an error is an AppError and optionally matches a status code.
func IsAppError(err error, codes ...int) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		if len(codes) == 0 {
			return appErr, true
		}
		for _, c := range codes {
			if appErr.Code == c {
				return appErr, true
			}
		}
	}
	return nil, false
}

// StatusCode extracts the HTTP status code from an error.
func StatusCode(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code
	}
	return http.StatusInternalServerError
}
