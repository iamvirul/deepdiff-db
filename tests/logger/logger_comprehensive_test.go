package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/logger"
)

func TestNew_DefaultValues(t *testing.T) {
	// Test with nil Output (should default to os.Stdout)
	log := logger.New(logger.Config{})
	if log == nil {
		t.Fatal("expected non-nil logger")
	}

	// Test with empty Format (should default to "text")
	var buf bytes.Buffer
	log = logger.New(logger.Config{
		Output: &buf,
		Format: "",
	})
	log.Info("test")
	if !strings.Contains(buf.String(), "level=INFO") {
		t.Error("expected text format by default")
	}
}

func TestNew_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: &buf,
	})

	log.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("expected JSON format, got: %s", output)
	}
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("expected message in JSON, got: %s", output)
	}
}

func TestNew_JSONFormatWithFileOutput(t *testing.T) {
	var consoleBuf bytes.Buffer
	var fileBuf bytes.Buffer

	log := logger.New(logger.Config{
		Level:      slog.LevelInfo,
		Format:     "json",
		Output:     &consoleBuf,
		FileOutput: &fileBuf,
	})

	log.Info("test message")

	// Both should have JSON format
	consoleOutput := consoleBuf.String()
	fileOutput := fileBuf.String()

	if !strings.Contains(consoleOutput, `"level":"INFO"`) {
		t.Error("expected JSON in console output")
	}
	if !strings.Contains(fileOutput, `"level":"INFO"`) {
		t.Error("expected JSON in file output")
	}
}

func TestNew_WithSource(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:      slog.LevelInfo,
		Format:     "text",
		Output:     &buf,
		WithSource: true,
	})

	log.Info("test")

	output := buf.String()
	// With source enabled, should contain file path
	if !strings.Contains(output, ".go") {
		t.Error("expected source location in output")
	}
}

func TestLogOperation_WithError(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:         slog.LevelInfo,
		Format:        "text",
		Output:        &buf,
		EnableMetrics: true,
	})

	ctx := context.Background()
	testErr := os.ErrNotExist

	err := log.LogOperation(ctx, "test_op", func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}

	output := buf.String()
	if !strings.Contains(output, "operation failed") {
		t.Errorf("expected 'operation failed' in output, got: %s", output)
	}
	if !strings.Contains(output, "error") {
		t.Errorf("expected error in output, got: %s", output)
	}

	// Check metrics were still recorded
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

func TestLogOperation_WithoutMetrics(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:         slog.LevelInfo,
		Format:        "text",
		Output:        &buf,
		EnableMetrics: false,
	})

	ctx := context.Background()
	err := log.LogOperation(ctx, "test_op", func() error {
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Metrics should be nil
	metrics := log.GetMetrics()
	if metrics != nil {
		t.Error("expected no metrics when disabled")
	}
}

func TestGetMetrics_MultipleOperations(t *testing.T) {
	log := logger.New(logger.Config{
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Record multiple operations
	_ = log.LogOperation(ctx, "op1", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	_ = log.LogOperation(ctx, "op2", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})
	_ = log.LogOperation(ctx, "op1", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	metrics := log.GetMetrics()
	if metrics == nil {
		t.Fatal("expected metrics to be collected")
	}

	if len(metrics) != 2 {
		t.Errorf("expected 2 unique operations, got %d", len(metrics))
	}

	op1, ok := metrics["op1"]
	if !ok {
		t.Fatal("expected op1 metric")
	}
	if op1.Count != 2 {
		t.Errorf("expected op1 count=2, got %d", op1.Count)
	}

	op2, ok := metrics["op2"]
	if !ok {
		t.Fatal("expected op2 metric")
	}
	if op2.Count != 1 {
		t.Errorf("expected op2 count=1, got %d", op2.Count)
	}
}

func TestPrintMetricsSummary(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Record some operations
	_ = log.LogOperation(ctx, "test_op1", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	_ = log.LogOperation(ctx, "test_op2", func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	log.PrintMetricsSummary(&buf)

	output := buf.String()
	if !strings.Contains(output, "Operation Metrics:") {
		t.Error("expected metrics header")
	}
	if !strings.Contains(output, "test_op1") {
		t.Error("expected test_op1 in summary")
	}
	if !strings.Contains(output, "test_op2") {
		t.Error("expected test_op2 in summary")
	}
	if !strings.Contains(output, "Count") {
		t.Error("expected Count column")
	}
	if !strings.Contains(output, "Total Time") {
		t.Error("expected Total Time column")
	}
}

func TestPrintMetricsSummary_Empty(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		EnableMetrics: true,
	})

	log.PrintMetricsSummary(&buf)

	if buf.Len() != 0 {
		t.Errorf("expected empty output, got: %s", buf.String())
	}
}

func TestPrintMetricsSummary_NoMetrics(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		EnableMetrics: false,
	})

	log.PrintMetricsSummary(&buf)

	if buf.Len() != 0 {
		t.Errorf("expected empty output when metrics disabled, got: %s", buf.String())
	}
}

func TestWith_MetricsSharing(t *testing.T) {
	log := logger.New(logger.Config{
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Create enhanced logger
	enhancedLog := log.With("table", "users")

	// Use both loggers
	_ = log.LogOperation(ctx, "op1", func() error { return nil })
	_ = enhancedLog.LogOperation(ctx, "op2", func() error { return nil })

	// Both should share the same metrics collector
	metrics := log.GetMetrics()
	if metrics == nil {
		t.Fatal("expected metrics")
	}

	// Both operations should be recorded
	if _, ok := metrics["op1"]; !ok {
		t.Error("expected op1 metric")
	}
	if _, ok := metrics["op2"]; !ok {
		t.Error("expected op2 metric")
	}
}

func TestWithDatabase(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New(logger.Config{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	enhancedLog := log.WithDatabase("mysql", "testdb")
	enhancedLog.Info("test")

	output := buf.String()
	if !strings.Contains(output, "driver=mysql") {
		t.Error("expected driver field")
	}
	if !strings.Contains(output, "database=testdb") {
		t.Error("expected database field")
	}
}

func TestRecordMetric_Concurrent(t *testing.T) {
	log := logger.New(logger.Config{
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Run multiple operations concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			_ = log.LogOperation(ctx, "concurrent_op", func() error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := log.GetMetrics()
	if metrics == nil {
		t.Fatal("expected metrics")
	}

	metric, ok := metrics["concurrent_op"]
	if !ok {
		t.Fatal("expected concurrent_op metric")
	}

	if metric.Count != 10 {
		t.Errorf("expected count=10, got %d", metric.Count)
	}
}

