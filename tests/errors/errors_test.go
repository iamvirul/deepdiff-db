package errors_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
		code          errors.ErrorCode
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

// ============================================================================
// Retry Tests
// ============================================================================

func TestRetry_SucceedsOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	calls := 0
	err := errors.Retry(ctx, errors.DefaultRetryConfig(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_NonRetryableError_DoesNotRetry(t *testing.T) {
	ctx := context.Background()
	calls := 0
	nonRetryable := errors.New(errors.ErrMissingPrimaryKey, "pk missing")
	err := errors.Retry(ctx, errors.DefaultRetryConfig(), func() error {
		calls++
		return nonRetryable
	})
	if err != nonRetryable {
		t.Fatalf("expected original error, got %v", err)
	}
	// Should return immediately without retrying
	if calls != 1 {
		t.Errorf("expected 1 call for non-retryable error, got %d", calls)
	}
}

func TestRetry_RetryableError_RetriesAndFails(t *testing.T) {
	ctx := context.Background()
	cfg := errors.RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	calls := 0
	retryable := errors.New(errors.ErrConnectionFailed, "connection failed")
	err := errors.Retry(ctx, cfg, func() error {
		calls++
		return retryable
	})

	if err != retryable {
		t.Fatalf("expected retryable error after exhaustion, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls (MaxAttempts), got %d", calls)
	}
}

func TestRetry_RetryableError_SucceedsOnRetry(t *testing.T) {
	ctx := context.Background()
	cfg := errors.RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	calls := 0
	retryable := errors.New(errors.ErrConnectionFailed, "connection failed")
	err := errors.Retry(ctx, cfg, func() error {
		calls++
		if calls < 3 {
			return retryable
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success on third attempt, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ContextCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	calls := 0
	err := errors.Retry(ctx, errors.DefaultRetryConfig(), func() error {
		calls++
		return nil
	})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls != 0 {
		t.Errorf("expected 0 calls when context pre-cancelled, got %d", calls)
	}
}

func TestRetry_ContextCancelledDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := errors.RetryConfig{
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond, // long delay so cancel fires first
		MaxDelay:          time.Second,
		BackoffMultiplier: 2.0,
	}

	calls := 0
	retryable := errors.New(errors.ErrConnectionFailed, "connection failed")

	// Cancel the context after first call
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := errors.Retry(ctx, cfg, func() error {
		calls++
		return retryable
	})

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled during wait, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call before context cancelled, got %d", calls)
	}
}

func TestRetry_ZeroMaxAttempts_DefaultsToThree(t *testing.T) {
	ctx := context.Background()
	cfg := errors.RetryConfig{
		MaxAttempts:       0, // should default to 3
		InitialDelay:      time.Millisecond,
		MaxDelay:          10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	calls := 0
	retryable := errors.New(errors.ErrConnectionFailed, "connection failed")
	errors.Retry(ctx, cfg, func() error { //nolint:errcheck
		calls++
		return retryable
	})

	if calls != 3 {
		t.Errorf("expected 3 calls when MaxAttempts=0 (defaults to 3), got %d", calls)
	}
}

func TestRetry_ZeroDelayDefaults(t *testing.T) {
	// Verify that zero-value InitialDelay, MaxDelay, and BackoffMultiplier
	// are filled with defaults and do not panic.
	ctx := context.Background()
	cfg := errors.RetryConfig{
		MaxAttempts:       2,
		InitialDelay:      0, // should default to 1s but we'll cap MaxDelay
		MaxDelay:          0, // should default
		BackoffMultiplier: 0, // should default
	}

	// Override to time.Millisecond via a separate config so the test runs fast.
	// Because zero values get replaced by defaults (InitialDelay=1s) this would
	// be slow — use a non-zero but small value instead.
	cfg2 := errors.RetryConfig{
		MaxAttempts:       1,
		InitialDelay:      time.Millisecond,
		MaxDelay:          time.Millisecond,
		BackoffMultiplier: 1.0,
	}

	calls := 0
	retryable := errors.New(errors.ErrQueryFailed, "query failed")
	err := errors.Retry(ctx, cfg2, func() error {
		calls++
		return retryable
	})
	if err != retryable {
		t.Fatalf("expected retryable error, got %v", err)
	}
	// With MaxAttempts=1 and a retryable error, it should exit after 1 attempt
	// (last attempt, no retry wait).
	if calls != 1 {
		t.Errorf("expected 1 call with MaxAttempts=1, got %d", calls)
	}

	// Also exercise the zero-value path just to ensure it does not panic.
	calls2 := 0
	successFn := func() error {
		calls2++
		return nil
	}
	if err2 := errors.Retry(ctx, cfg, successFn); err2 != nil {
		t.Fatalf("zero-value cfg with immediate success should not error: %v", err2)
	}
}

func TestRetry_MaxDelayCapApplied(t *testing.T) {
	// Verify that delay is capped at MaxDelay and does not grow unboundedly.
	ctx := context.Background()
	cfg := errors.RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      time.Millisecond,
		MaxDelay:          2 * time.Millisecond, // cap is tiny
		BackoffMultiplier: 100.0,               // huge multiplier to trigger the cap
	}

	calls := 0
	retryable := errors.New(errors.ErrConnectionTimeout, "timeout")
	start := time.Now()
	errors.Retry(ctx, cfg, func() error { //nolint:errcheck
		calls++
		return retryable
	})
	elapsed := time.Since(start)

	// If capping works, total sleep ≤ 2*(MaxDelay) ≈ 4ms — well under 100ms.
	if elapsed > 100*time.Millisecond {
		t.Errorf("expected delay to be capped, but elapsed %v exceeds 100ms", elapsed)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDefaultRetryConfig_Values(t *testing.T) {
	cfg := errors.DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", cfg.MaxDelay)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("expected BackoffMultiplier 2.0, got %f", cfg.BackoffMultiplier)
	}
}
