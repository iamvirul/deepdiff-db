package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

func TestError_Format(t *testing.T) {
	originalErr := fmt.Errorf("underlying error")
	err := errors.Wrap(originalErr, errors.ErrConnectionFailed, "connection failed").
		With("host", "localhost").
		WithSuggestion("check network")

	// Test %v (standard format)
	standard := fmt.Sprintf("%v", err)
	if !strings.Contains(standard, "connection failed") {
		t.Errorf("expected message in standard format, got: %s", standard)
	}

	// Test %+v (detailed format)
	detailed := fmt.Sprintf("%+v", err)
	if !strings.Contains(detailed, "Error:") {
		t.Errorf("expected detailed format header, got: %s", detailed)
	}
	if !strings.Contains(detailed, "Category:") {
		t.Errorf("expected category in detailed format, got: %s", detailed)
	}
	if !strings.Contains(detailed, "Context:") {
		t.Errorf("expected context in detailed format, got: %s", detailed)
	}
	if !strings.Contains(detailed, "Suggestions:") {
		t.Errorf("expected suggestions in detailed format, got: %s", detailed)
	}

	// Test %s (string format)
	stringFmt := fmt.Sprintf("%s", err)
	if !strings.Contains(stringFmt, "connection failed") {
		t.Errorf("expected message in string format, got: %s", stringFmt)
	}
}

func TestError_Error_NoCause(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	errorString := err.Error()
	if !strings.Contains(errorString, "connection failed") {
		t.Errorf("expected message in error string, got: %s", errorString)
	}
	if strings.Contains(errorString, ":") {
		t.Error("expected no cause separator when cause is nil")
	}
}

func TestError_With_NilContext(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "test")
	// Manually set context to nil to test initialization
	err.Context = nil

	enhanced := err.With("key", "value")
	if enhanced.Context == nil {
		t.Error("expected context to be initialized")
	}
	if enhanced.Context["key"] != "value" {
		t.Error("expected key-value to be set")
	}
}

func TestError_DebugString_AllFields(t *testing.T) {
	originalErr := fmt.Errorf("underlying error")
	err := errors.Wrap(originalErr, errors.ErrConnectionFailed, "connection failed").
		With("host", "localhost").
		With("port", 3306).
		WithSuggestion("check network").
		WithSuggestion("verify credentials").
		WithStackTrace(0)

	debugStr := err.DebugString()

	// Check all sections are present
	expectedSections := []string{
		"Error:",
		"Category:",
		"Caused by:",
		"Context:",
		"host = localhost",
		"port = 3306",
		"Suggestions:",
		"check network",
		"verify credentials",
		"Stack trace:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(debugStr, section) {
			t.Errorf("expected debug string to contain %q, got:\n%s", section, debugStr)
		}
	}
}

func TestError_DebugString_NoCause(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	debugStr := err.DebugString()
	if strings.Contains(debugStr, "Caused by:") {
		t.Error("expected no 'Caused by' section when cause is nil")
	}
}

func TestError_DebugString_NoContext(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	debugStr := err.DebugString()
	if strings.Contains(debugStr, "Context:") {
		t.Error("expected no 'Context' section when context is empty")
	}
}

func TestError_DebugString_NoSuggestions(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	debugStr := err.DebugString()
	if strings.Contains(debugStr, "Suggestions:") {
		t.Error("expected no 'Suggestions' section when suggestions are empty")
	}
}

func TestError_DebugString_NoStackTrace(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	debugStr := err.DebugString()
	if strings.Contains(debugStr, "Stack trace:") {
		t.Error("expected no 'Stack trace' section when stack trace is empty")
	}
}

func TestWithStackTrace_Comprehensive(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "test").
		WithStackTrace(0)

	if len(err.StackTrace) == 0 {
		t.Error("expected stack trace to be captured")
	}

	// Verify stack trace contains function names
	hasFunction := false
	for _, frame := range err.StackTrace {
		if strings.Contains(frame, "TestWithStackTrace") {
			hasFunction = true
			break
		}
	}
	if !hasFunction {
		t.Error("expected stack trace to contain test function")
	}
}

func TestCaptureStackTrace_SkipFrames(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "test").
		WithStackTrace(5) // Skip more frames

	if len(err.StackTrace) == 0 {
		t.Error("expected stack trace even with skip")
	}
}

func TestError_Format_UnknownVerb(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "test")

	// Test with unknown verb (should not panic, may return empty or default format)
	result := fmt.Sprintf("%d", err)
	_ = result // Just ensure it doesn't panic
}

func TestWrap_NilError(t *testing.T) {
	wrapped := errors.Wrap(nil, errors.ErrConnectionFailed, "test")
	if wrapped != nil {
		t.Error("wrapping nil should return nil")
	}
}

func TestError_Unwrap_Nil(t *testing.T) {
	var err *errors.Error
	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Error("unwrap of nil error should return nil")
	}
}

func TestError_IsRetryable_Nil(t *testing.T) {
	var err *errors.Error
	if err.IsRetryable() {
		t.Error("nil error should not be retryable")
	}
}

func TestError_Format_WithNil(t *testing.T) {
	var err *errors.Error

	// These should not panic
	_ = fmt.Sprintf("%v", err)
	_ = fmt.Sprintf("%+v", err)
	_ = fmt.Sprintf("%s", err)
}

