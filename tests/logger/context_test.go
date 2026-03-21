package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/logger"
)

func TestToContext(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	ctx := logger.ToContext(context.Background(), log)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Should be able to retrieve the same logger
	retrieved := logger.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected to retrieve logger from context")
	}

	// Log with retrieved logger
	retrieved.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("expected logger to work after retrieval from context")
	}
}

func TestFromContext_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	ctx := logger.ToContext(context.Background(), log)
	retrieved := logger.FromContext(ctx)

	retrieved.Info("test message")
	if !strings.Contains(buf.String(), "test message") {
		t.Error("expected message from retrieved logger")
	}
}

func TestFromContext_WithoutLogger(t *testing.T) {
	// Should return default logger, not nil
	ctx := context.Background()
	log := logger.FromContext(ctx)

	if log == nil {
		t.Fatal("expected non-nil default logger")
	}

	// Should be able to use default logger without panic
	log.Info("test message")
}

func TestFromContext_NilContext(t *testing.T) {
	// Should handle context without logger gracefully
	ctx := context.TODO()
	log := logger.FromContext(ctx)

	if log == nil {
		t.Fatal("expected non-nil default logger for context without logger")
	}

	// Should be able to use default logger
	log.Info("test message")
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	ctx := logger.ToContext(context.Background(), log)
	ctx = logger.WithFields(ctx, "table", "users", "operation", "hash")

	enhancedLog := logger.FromContext(ctx)
	enhancedLog.Info("processing")

	output := buf.String()
	if !strings.Contains(output, "table=users") {
		t.Errorf("expected output to contain table=users, got: %s", output)
	}
	if !strings.Contains(output, "operation=hash") {
		t.Errorf("expected output to contain operation=hash, got: %s", output)
	}
}

func TestContextChaining(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	// Chain context enhancements
	ctx := context.Background()
	ctx = logger.ToContext(ctx, log)
	ctx = logger.WithFields(ctx, "database", "testdb")
	ctx = logger.WithFields(ctx, "table", "users")

	enhancedLog := logger.FromContext(ctx)
	enhancedLog.Info("test")

	output := buf.String()
	if !strings.Contains(output, "database=testdb") {
		t.Error("expected database field in output")
	}
	if !strings.Contains(output, "table=users") {
		t.Error("expected table field in output")
	}
}

func TestSetDefaultOutput(t *testing.T) {
	var buf bytes.Buffer

	// Set default output to our buffer
	logger.SetDefaultOutput(&buf)

	// Get default logger (should use our output)
	ctx := context.Background()
	log := logger.FromContext(ctx)

	log.Info("test message")

	if !strings.Contains(buf.String(), "test message") {
		t.Error("expected default logger to use SetDefaultOutput")
	}
}
