package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// loggerKeyType is a private type for context keys to avoid collisions.
type loggerKeyType struct{}

// loggerKey is the context key for storing Logger instances.
var loggerKey = loggerKeyType{}

// defaultLogger is the fallback logger used when no logger is found in context.
var defaultLogger = &Logger{
	slog: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})),
}

// ToContext adds a logger to the given context and returns the new context.
// This allows the logger to be passed through function call chains without
// explicit parameters.
//
// Example:
//
//	logger := logger.New(logger.Config{...})
//	ctx = logger.ToContext(ctx, logger)
func ToContext(ctx context.Context, l *Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext retrieves the logger from the given context.
// If no logger is found, it returns a default logger that writes to stdout
// at INFO level. This ensures logging always works even if context setup
// is forgotten.
//
// Example:
//
//	logger := logger.FromContext(ctx)
//	logger.Info("operation started")
func FromContext(ctx context.Context) *Logger {
	if ctx == nil {
		return defaultLogger
	}

	if l, ok := ctx.Value(loggerKey).(*Logger); ok && l != nil {
		return l
	}

	return defaultLogger
}

// WithFields adds structured fields to the logger in the context and returns
// a new context with the enhanced logger. The fields are key-value pairs that
// will be included in all subsequent log entries from this context.
//
// Example:
//
//	ctx = logger.WithFields(ctx, "table", "users", "operation", "hash")
//	logger.FromContext(ctx).Info("processing") // Will include table and operation fields
func WithFields(ctx context.Context, fields ...any) context.Context {
	l := FromContext(ctx)
	enhanced := l.With(fields...)
	return ToContext(ctx, enhanced)
}

// SetDefaultOutput changes the output destination for the default logger.
// This is useful for testing or when you need to redirect default logs.
func SetDefaultOutput(w io.Writer) {
	defaultLogger = &Logger{
		slog: slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
	}
}
