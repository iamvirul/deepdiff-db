package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/logger"
)

func TestNew(t *testing.T) {
	tests := []struct{
		name   string
		config logger.Config
		want   string // Substring to find in log output
	}{
		{
			name: "text_format_info_level",
			config: logger.Config{
				Level:  slog.LevelInfo,
				Format: "text",
			},
			want: "level=INFO",
		},
		{
			name: "json_format_debug_level",
			config: logger.Config{
				Level:  slog.LevelDebug,
				Format: "json",
			},
			want: `"level":"DEBUG"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.config.Output = &buf

			log := logger.New(tt.config)
			log.Debug("test message", "key", "value")

			if tt.config.Level == slog.LevelInfo && strings.Contains(buf.String(), "test message") {
				t.Error("DEBUG message should not appear at INFO level")
			}

			if tt.config.Level == slog.LevelDebug {
				if !strings.Contains(buf.String(), tt.want) {
					t.Errorf("expected output to contain %q, got: %s", tt.want, buf.String())
				}
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelDebug,
		Format: "text",
		Output: &buf,
	})

	tests := []struct {
		name    string
		logFunc func(string, ...any)
		message string
		level   string
	}{
		{
			name:    "debug_level",
			logFunc: log.Debug,
			message: "debug message",
			level:   "level=DEBUG",
		},
		{
			name:    "info_level",
			logFunc: log.Info,
			message: "info message",
			level:   "level=INFO",
		},
		{
			name:    "warn_level",
			logFunc: log.Warn,
			message: "warn message",
			level:   "level=WARN",
		},
		{
			name:    "error_level",
			logFunc: log.Error,
			message: "error message",
			level:   "level=ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message, "key", "value")

			output := buf.String()
			if !strings.Contains(output, tt.level) {
				t.Errorf("expected output to contain %q, got: %s", tt.level, output)
			}
			if !strings.Contains(output, tt.message) {
				t.Errorf("expected output to contain %q, got: %s", tt.message, output)
			}
		})
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	baseLog := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	enhancedLog := baseLog.With("table", "users", "operation", "hash")
	enhancedLog.Info("processing")

	output := buf.String()
	if !strings.Contains(output, "table=users") {
		t.Errorf("expected output to contain table=users, got: %s", output)
	}
	if !strings.Contains(output, "operation=hash") {
		t.Errorf("expected output to contain operation=hash, got: %s", output)
	}
}

func TestWithHelpers(t *testing.T) {
	var buf bytes.Buffer
	baseLog := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	tests := []struct {
		name      string
		enhanceFunc func(*logger.Logger) *logger.Logger
		wantField string
	}{
		{
			name: "with_table",
			enhanceFunc: func(l *logger.Logger) *logger.Logger {
				return l.WithTable("users")
			},
			wantField: "table=users",
		},
		{
			name: "with_operation",
			enhanceFunc: func(l *logger.Logger) *logger.Logger {
				return l.WithOperation("hash")
			},
			wantField: "operation=hash",
		},
		{
			name: "with_database",
			enhanceFunc: func(l *logger.Logger) *logger.Logger {
				return l.WithDatabase("mysql", "testdb")
			},
			wantField: "driver=mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			enhanced := tt.enhanceFunc(baseLog)
			enhanced.Info("test message")

			output := buf.String()
			if !strings.Contains(output, tt.wantField) {
				t.Errorf("expected output to contain %q, got: %s", tt.wantField, output)
			}
		})
	}
}

func TestLogOperation(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:         slog.LevelInfo,
		Format:        "text",
		Output:        &buf,
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Successful operation
	err := log.LogOperation(ctx, "test_op", func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "operation=test_op") {
		t.Errorf("expected output to contain operation name, got: %s", output)
	}
	if !strings.Contains(output, "operation complete") {
		t.Errorf("expected output to contain 'operation complete', got: %s", output)
	}
	if !strings.Contains(output, "duration_ms") {
		t.Errorf("expected output to contain duration_ms, got: %s", output)
	}

	// Check metrics
	metrics := log.GetMetrics()
	if metrics == nil {
		t.Error("expected metrics to be collected")
	}
	if metric, ok := metrics["test_op"]; !ok {
		t.Error("expected test_op metric to exist")
	} else if metric.Count != 1 {
		t.Errorf("expected count=1, got: %d", metric.Count)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // Default to info
		{"", slog.LevelInfo},         // Default to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := logger.ParseLevel(tt.input)
			if got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFileOutput(t *testing.T) {
	var consoleBuf bytes.Buffer
	var fileBuf bytes.Buffer

	log := logger.New(logger.Config{
		Level:      slog.LevelInfo,
		Format:     "text",
		Output:     &consoleBuf,
		FileOutput: &fileBuf,
	})

	log.Info("test message")

	// Should write to both outputs
	if !strings.Contains(consoleBuf.String(), "test message") {
		t.Error("expected message in console output")
	}
	if !strings.Contains(fileBuf.String(), "test message") {
		t.Error("expected message in file output")
	}
}
