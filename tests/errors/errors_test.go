package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

func TestNew(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "could not connect")

	if err == nil {
		t.Fatal("expected non-nil error")
	}

	if err.Code != errors.ErrConnectionFailed {
		t.Errorf("expected code %s, got %s", errors.ErrConnectionFailed, err.Code)
	}

	if err.Message != "could not connect" {
		t.Errorf("expected message 'could not connect', got %q", err.Message)
	}

	if err.Context == nil {
		t.Error("expected non-nil context map")
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("dial tcp: connection refused")
	wrappedErr := errors.Wrap(originalErr, errors.ErrConnectionFailed, "database connection failed")

	if wrappedErr.Code != errors.ErrConnectionFailed {
		t.Errorf("expected code %s, got %s", errors.ErrConnectionFailed, wrappedErr.Code)
	}

	if wrappedErr.Message != "database connection failed" {
		t.Errorf("expected message 'database connection failed', got %q", wrappedErr.Message)
	}

	if wrappedErr.Cause != originalErr {
		t.Error("expected cause to be the original error")
	}
}

func TestWrapNil(t *testing.T) {
	wrappedErr := errors.Wrap(nil, errors.ErrConnectionFailed, "test")
	if wrappedErr != nil {
		t.Error("wrapping nil error should return nil")
	}
}

func TestWith(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		With("host", "localhost").
		With("port", 3306)

	if err.Context["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", err.Context["host"])
	}

	if err.Context["port"] != 3306 {
		t.Errorf("expected port=3306, got %v", err.Context["port"])
	}
}

func TestWithSuggestion(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		WithSuggestion("Check network connectivity").
		WithSuggestion("Verify database is running")

	if len(err.Suggestions) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(err.Suggestions))
	}

	if err.Suggestions[0] != "Check network connectivity" {
		t.Errorf("unexpected first suggestion: %s", err.Suggestions[0])
	}
}

func TestError(t *testing.T) {
	originalErr := fmt.Errorf("connection refused")
	err := errors.Wrap(originalErr, errors.ErrConnectionFailed, "database connection failed")

	errorString := err.Error()

	if !strings.Contains(errorString, string(errors.ErrConnectionFailed)) {
		t.Errorf("expected error string to contain code, got: %s", errorString)
	}

	if !strings.Contains(errorString, "database connection failed") {
		t.Errorf("expected error string to contain message, got: %s", errorString)
	}

	if !strings.Contains(errorString, "connection refused") {
		t.Errorf("expected error string to contain cause, got: %s", errorString)
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := fmt.Errorf("connection refused")
	wrappedErr := errors.Wrap(originalErr, errors.ErrConnectionFailed, "connection failed")

	unwrapped := wrappedErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("expected Unwrap to return original error")
	}
}

func TestDebugString(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		With("host", "localhost").
		With("port", 3306).
		WithSuggestion("Check network")

	debugStr := err.DebugString()

	expectedContents := []string{
		"Error:",
		string(errors.ErrConnectionFailed),
		"connection failed",
		"Category: Connection",
		"Context:",
		"host = localhost",
		"port = 3306",
		"Suggestions:",
		"Check network",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(debugStr, expected) {
			t.Errorf("expected debug string to contain %q, got:\n%s", expected, debugStr)
		}
	}
}

func TestWithStackTrace(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		WithStackTrace(0)

	if len(err.StackTrace) == 0 {
		t.Error("expected stack trace to be captured")
	}

	debugStr := err.DebugString()
	if !strings.Contains(debugStr, "Stack trace:") {
		t.Error("expected debug string to contain stack trace")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code        errors.ErrorCode
		wantRetryable bool
	}{
		{errors.ErrConnectionFailed, true},
		{errors.ErrConnectionTimeout, true},
		{errors.ErrQueryFailed, true},
		{errors.ErrTransactionFailed, true},
		{errors.ErrMissingPrimaryKey, false},
		{errors.ErrSchemaDrift, false},
		{errors.ErrConfigInvalid, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if tt.code.IsRetryable() != tt.wantRetryable {
				t.Errorf("IsRetryable() = %v, want %v", tt.code.IsRetryable(), tt.wantRetryable)
			}

			// Also test via Error.IsRetryable()
			err := errors.New(tt.code, "test")
			if err.IsRetryable() != tt.wantRetryable {
				t.Errorf("Error.IsRetryable() = %v, want %v", err.IsRetryable(), tt.wantRetryable)
			}
		})
	}
}

func TestErrorCodeCategory(t *testing.T) {
	tests := []struct {
		code     errors.ErrorCode
		category string
	}{
		{errors.ErrConnectionFailed, "Connection"},
		{errors.ErrSchemaDrift, "Schema"},
		{errors.ErrHashingFailed, "Data"},
		{errors.ErrPackGeneration, "Migration"},
		{errors.ErrCheckpointRead, "Checkpoint"},
		{errors.ErrConfigInvalid, "Configuration"},
		{errors.ErrOutOfMemory, "System"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := tt.code.Category()
			if got != tt.category {
				t.Errorf("Category() = %q, want %q", got, tt.category)
			}
		})
	}
}

func TestErrorNilHandling(t *testing.T) {
	var err *errors.Error

	// These should all handle nil gracefully without panicking
	_ = err.Error()
	_ = err.Unwrap()
	_ = err.DebugString()
	_ = err.IsRetryable()

	err = err.With("key", "value")
	if err != nil {
		t.Error("With on nil error should return nil")
	}

	err = err.WithSuggestion("test")
	if err != nil {
		t.Error("WithSuggestion on nil error should return nil")
	}

	err = err.WithStackTrace(0)
	if err != nil {
		t.Error("WithStackTrace on nil error should return nil")
	}
}
