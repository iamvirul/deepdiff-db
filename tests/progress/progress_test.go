package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

func TestNewManager(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  os.Stderr,
	})

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if !mgr.IsEnabled() {
		t.Error("manager should be enabled")
	}
}

func TestNewManagerDisabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: false,
	})

	if mgr.IsEnabled() {
		t.Error("manager should be disabled")
	}
}

func TestNewManagerDefaultOutput(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  nil, // Should default to os.Stderr
	})

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestStartBar(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if bar == nil {
		t.Fatal("StartBar returned nil")
	}

	if bar.IsComplete() {
		t.Error("new bar should not be complete")
	}
}

func TestStartBarDisabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: false,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if bar == nil {
		t.Fatal("StartBar returned nil")
	}

	// When disabled, bar.pb is nil, but Add should still work for metrics
	if err := bar.Add(10); err != nil {
		t.Errorf("Add should work even when disabled: %v", err)
	}

	if bar.IsComplete() {
		t.Error("new bar should not be complete")
	}
}

func TestStartSpinner(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartSpinner(ctx, "test")

	if bar == nil {
		t.Fatal("StartSpinner returned nil")
	}

	if bar.IsComplete() {
		t.Error("new spinner should not be complete")
	}
}

func TestGetBar(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar1 := mgr.StartBar(ctx, "test1", 100)

	retrieved := mgr.GetBar("test1")
	if retrieved != bar1 {
		t.Error("GetBar should return the same bar")
	}

	if mgr.GetBar("nonexistent") != nil {
		t.Error("GetBar should return nil for nonexistent bar")
	}
}

func TestFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: true,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if err := bar.Add(50); err != nil {
		t.Errorf("Add failed: %v", err)
	}

	mgr.Finish()

	if !bar.IsComplete() {
		t.Error("bar should be complete after Finish")
	}
}

func TestBarAdd(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if err := bar.Add(10); err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if err := bar.Add(20); err != nil {
		t.Errorf("Add failed: %v", err)
	}
}

func TestBarAddAfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish() // Ignore error - bar is finishing

	// Add after finish should not error
	if err := bar.Add(10); err != nil {
		t.Errorf("Add after finish should not error: %v", err)
	}
}

func TestBarSet(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if err := bar.Set(50); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	if err := bar.Set(100); err != nil {
		t.Errorf("Set failed: %v", err)
	}
}

func TestBarSetAfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish() // Ignore error - bar is finishing

	// Set after finish should not error
	if err := bar.Set(50); err != nil {
		t.Errorf("Set after finish should not error: %v", err)
	}
}

func TestBarDescribe(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	bar.Describe("new description")
	// No error means success
}

func TestBarDescribeAfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish() // Ignore error - bar is finishing
	bar.Describe("should not update")
	// Should not error
}

func TestBarThroughput(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Add(50) // Ignore error - progress update
	time.Sleep(10 * time.Millisecond)

	throughput := bar.Throughput()
	if throughput < 0 {
		t.Error("throughput should be non-negative")
	}
}

func TestBarDuration(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	time.Sleep(10 * time.Millisecond)

	duration := bar.Duration()
	if duration < 0 {
		t.Error("duration should be non-negative")
	}
}

func TestBarString(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Add(50) // Ignore error - progress update
	str := bar.String()

	if !strings.Contains(str, "test") {
		t.Error("String should contain bar name")
	}

	if !strings.Contains(str, "50") {
		t.Error("String should contain current value")
	}
}

func TestBarStringSpinner(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartSpinner(ctx, "test")

	_ = bar.Add(50) // Ignore error - progress update
	str := bar.String()

	if !strings.Contains(str, "test") {
		t.Error("String should contain bar name")
	}
}

func TestContext(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	ctx = progress.ToContext(ctx, mgr)

	retrieved := progress.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("FromContext returned nil")
	}

	if retrieved != mgr {
		t.Error("retrieved manager should be the same instance")
	}
}

func TestFromContextNil(t *testing.T) {
	if progress.FromContext(context.TODO()) != nil {
		t.Error("FromContext with context without manager should return nil")
	}

	ctx := context.Background()
	if progress.FromContext(ctx) != nil {
		t.Error("FromContext with context without manager should return nil")
	}
}

func TestGetMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: true,
	})

	metrics := mgr.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics should return metrics when enabled")
	}
}

func TestGetMetricsDisabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: false,
	})

	metrics := mgr.GetMetrics()
	if metrics != nil {
		t.Error("GetMetrics should return nil when disabled")
	}
}

func TestNewMetrics(t *testing.T) {
	metrics := progress.NewMetrics()
	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

func TestMetricsRecord(t *testing.T) {
	metrics := progress.NewMetrics()

	metrics.Record("test", 1*time.Second, 100)

	op := metrics.Get("test")
	if op == nil {
		t.Fatal("Get returned nil")
	}

	if op.Name != "test" {
		t.Errorf("expected name 'test', got %s", op.Name)
	}

	if op.Duration != 1*time.Second {
		t.Errorf("expected duration 1s, got %v", op.Duration)
	}

	if op.RowsProcessed != 100 {
		t.Errorf("expected 100 rows, got %d", op.RowsProcessed)
	}

	if op.Throughput <= 0 {
		t.Error("throughput should be positive")
	}
}

func TestMetricsRecordWithDetails(t *testing.T) {
	metrics := progress.NewMetrics()

	metrics.RecordWithDetails("test", 1*time.Second, 100, 50, 10.5)

	op := metrics.Get("test")
	if op == nil {
		t.Fatal("Get returned nil")
	}

	if op.QueryCount != 50 {
		t.Errorf("expected 50 queries, got %d", op.QueryCount)
	}

	if op.MemoryMB != 10.5 {
		t.Errorf("expected 10.5 MB, got %f", op.MemoryMB)
	}
}

func TestMetricsGetNonexistent(t *testing.T) {
	metrics := progress.NewMetrics()

	if metrics.Get("nonexistent") != nil {
		t.Error("Get should return nil for nonexistent operation")
	}
}

func TestMetricsSummary(t *testing.T) {
	metrics := progress.NewMetrics()

	// Empty metrics should return empty string
	summary := metrics.Summary()
	if summary != "" {
		t.Error("Summary should return empty string for no metrics")
	}

	// Add some metrics
	metrics.Record("op1", 1*time.Second, 100)
	metrics.Record("op2", 2*time.Second, 200)

	summary = metrics.Summary()
	if summary == "" {
		t.Error("Summary should not be empty with metrics")
	}

	if !strings.Contains(summary, "op1") {
		t.Error("Summary should contain op1")
	}

	if !strings.Contains(summary, "op2") {
		t.Error("Summary should contain op2")
	}

	if !strings.Contains(summary, "TOTAL") {
		t.Error("Summary should contain TOTAL")
	}
}

func TestMetricsThroughputZero(t *testing.T) {
	metrics := progress.NewMetrics()

	// Record with zero duration
	metrics.Record("test", 0, 100)

	op := metrics.Get("test")
	if op == nil {
		t.Fatal("Get returned nil")
	}

	if op.Throughput != 0 {
		t.Errorf("expected throughput 0 for zero duration, got %f", op.Throughput)
	}
}

func TestMetricsConcurrent(t *testing.T) {
	metrics := progress.NewMetrics()

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			metrics.Record("op", time.Duration(id)*time.Second, int64(id*10))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic
	summary := metrics.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
}

func TestManagerWithMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: true,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Add(50) // Ignore error - progress update
	time.Sleep(10 * time.Millisecond)

	mgr.Finish()

	metrics := mgr.GetMetrics()
	if metrics == nil {
		t.Fatal("metrics should not be nil")
	}

	op := metrics.Get("test")
	if op == nil {
		t.Fatal("operation metrics should be recorded")
	}

	if op.RowsProcessed != 50 {
		t.Errorf("expected 50 rows, got %d", op.RowsProcessed)
	}
}

