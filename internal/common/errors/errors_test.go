package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := NewAppError(http.StatusNotFound, "not found")
	if err.Error() != "not found" {
		t.Errorf("expected 'not found', got '%s'", err.Error())
	}
}

func TestAppError_Wrap(t *testing.T) {
	cause := errors.New("cause")
	err := ErrNotFound.Wrap(cause)
	if err.Error() != "resource not found: cause" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected to be ErrNotFound")
	}
}

func TestIsAppError(t *testing.T) {
	err := NewAppError(http.StatusTeapot, "teapot")
	appErr, ok := IsAppError(err, http.StatusTeapot)
	if !ok {
		t.Error("expected to match")
	}
	if appErr.Code != http.StatusTeapot {
		t.Errorf("expected code %d, got %d", http.StatusTeapot, appErr.Code)
	}

	_, ok = IsAppError(err, http.StatusNotFound)
	if ok {
		t.Error("expected not to match wrong code")
	}
}

func TestStatusCode(t *testing.T) {
	if c := StatusCode(ErrNotFound); c != http.StatusNotFound {
		t.Errorf("expected 404, got %d", c)
	}
	if c := StatusCode(errors.New("plain")); c != http.StatusInternalServerError {
		t.Errorf("expected 500 for plain error, got %d", c)
	}
}
