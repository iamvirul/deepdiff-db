package errors

import (
	"context"
	"errors"
	"time"
)

// RetryConfig holds configuration for retry logic.
type RetryConfig struct {
	// MaxAttempts is the maximum number of retry attempts (default: 3).
	MaxAttempts int

	// InitialDelay is the initial delay before first retry (default: 1 second).
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries (default: 30 seconds).
	MaxDelay time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff (default: 2.0).
	BackoffMultiplier float64
}

// DefaultRetryConfig returns a default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// Retry executes a function with retry logic for retryable errors.
// It will retry up to MaxAttempts times with exponential backoff if the error is retryable.
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2.0
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		var enhancedErr *Error
		if errors.As(err, &enhancedErr) && enhancedErr.IsRetryable() {
			// Don't retry on last attempt
			if attempt < cfg.MaxAttempts-1 {
				// Wait before retry
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}

				// Exponential backoff
				delay = time.Duration(float64(delay) * cfg.BackoffMultiplier)
				if delay > cfg.MaxDelay {
					delay = cfg.MaxDelay
				}
				continue
			}
		}

		// Non-retryable error or last attempt
		return err
	}

	return lastErr
}

