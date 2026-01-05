package main

import (
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

func TestNewMetrics_Comprehensive(t *testing.T) {
	m := progress.NewMetrics()
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}
	if m.Operations == nil {
		t.Error("expected non-nil operations map")
	}
}

func TestMetrics_Record(t *testing.T) {
	m := progress.NewMetrics()

	m.Record("test_op", 100*time.Millisecond, 1000)

	metric := m.Get("test_op")
	if metric == nil {
		t.Fatal("expected metric to be recorded")
	}

	if metric.Name != "test_op" {
		t.Errorf("expected name 'test_op', got: %s", metric.Name)
	}
	if metric.Duration != 100*time.Millisecond {
		t.Errorf("expected duration 100ms, got: %v", metric.Duration)
	}
	if metric.RowsProcessed != 1000 {
		t.Errorf("expected rows 1000, got: %d", metric.RowsProcessed)
	}
	if metric.Throughput <= 0 {
		t.Error("expected positive throughput")
	}
}

func TestMetrics_RecordWithDetails(t *testing.T) {
	m := progress.NewMetrics()

	m.RecordWithDetails("test_op", 100*time.Millisecond, 1000, 50, 128.5)

	metric := m.Get("test_op")
	if metric == nil {
		t.Fatal("expected metric to be recorded")
	}

	if metric.QueryCount != 50 {
		t.Errorf("expected query count 50, got: %d", metric.QueryCount)
	}
	if metric.MemoryMB != 128.5 {
		t.Errorf("expected memory 128.5MB, got: %f", metric.MemoryMB)
	}
}

func TestMetrics_Record_ZeroDuration(t *testing.T) {
	m := progress.NewMetrics()

	m.Record("test_op", 0, 1000)

	metric := m.Get("test_op")
	if metric == nil {
		t.Fatal("expected metric to be recorded")
	}

	if metric.Throughput != 0 {
		t.Errorf("expected zero throughput for zero duration, got: %f", metric.Throughput)
	}
}

func TestMetrics_Record_MemoryAuto(t *testing.T) {
	m := progress.NewMetrics()

	// Record without memory, should auto-calculate
	m.RecordWithDetails("test_op", 100*time.Millisecond, 1000, 0, 0)

	metric := m.Get("test_op")
	if metric == nil {
		t.Fatal("expected metric to be recorded")
	}

	// Memory should be auto-calculated (non-zero)
	if metric.MemoryMB <= 0 {
		t.Error("expected auto-calculated memory to be positive")
	}
}

func TestMetrics_Get_NotFound(t *testing.T) {
	m := progress.NewMetrics()

	metric := m.Get("nonexistent")
	if metric != nil {
		t.Error("expected nil for nonexistent metric")
	}
}

func TestMetrics_Get_Copy(t *testing.T) {
	m := progress.NewMetrics()

	m.Record("test_op", 100*time.Millisecond, 1000)

	metric1 := m.Get("test_op")
	metric2 := m.Get("test_op")

	// Should be different instances (copies)
	if metric1 == metric2 {
		t.Error("expected different instances")
	}

	// But same values
	if metric1.Name != metric2.Name {
		t.Error("expected same name")
	}
}

func TestMetrics_Summary_Empty(t *testing.T) {
	m := progress.NewMetrics()

	summary := m.Summary()
	if summary != "" {
		t.Errorf("expected empty summary, got: %s", summary)
	}
}

func TestMetrics_Summary_SingleOperation(t *testing.T) {
	m := progress.NewMetrics()

	m.Record("test_op", 100*time.Millisecond, 1000)

	summary := m.Summary()
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	expectedContents := []string{
		"Performance Metrics:",
		"Operation",
		"Duration",
		"Rows",
		"Throughput",
		"Memory",
		"Queries",
		"test_op",
		"TOTAL",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(summary, expected) {
			t.Errorf("expected summary to contain %q, got:\n%s", expected, summary)
		}
	}
}

func TestMetrics_Summary_MultipleOperations(t *testing.T) {
	m := progress.NewMetrics()

	m.Record("op1", 100*time.Millisecond, 1000)
	m.Record("op2", 200*time.Millisecond, 2000)
	m.RecordWithDetails("op3", 50*time.Millisecond, 500, 25, 64.0)

	summary := m.Summary()
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	// Check all operations are present
	if !strings.Contains(summary, "op1") {
		t.Error("expected op1 in summary")
	}
	if !strings.Contains(summary, "op2") {
		t.Error("expected op2 in summary")
	}
	if !strings.Contains(summary, "op3") {
		t.Error("expected op3 in summary")
	}

	// Check totals
	if !strings.Contains(summary, "TOTAL") {
		t.Error("expected TOTAL in summary")
	}
}

func TestMetrics_Summary_Concurrent(t *testing.T) {
	m := progress.NewMetrics()

	// Record metrics concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			m.Record("concurrent_op", 10*time.Millisecond, 100)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	summary := m.Summary()
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}

	if !strings.Contains(summary, "concurrent_op") {
		t.Error("expected concurrent_op in summary")
	}
}

func TestTruncateName(t *testing.T) {
	m := progress.NewMetrics()

	// Test with long name
	longName := strings.Repeat("a", 50)
	m.Record(longName, 100*time.Millisecond, 1000)

	summary := m.Summary()
	if !strings.Contains(summary, "...") {
		t.Error("expected truncated name with ...")
	}
}

func TestMetrics_Record_UpdateExisting(t *testing.T) {
	m := progress.NewMetrics()

	// Record same operation twice
	m.Record("test_op", 100*time.Millisecond, 1000)
	m.Record("test_op", 200*time.Millisecond, 2000)

	metric := m.Get("test_op")
	if metric == nil {
		t.Fatal("expected metric")
	}

	// Should have latest values
	if metric.Duration != 200*time.Millisecond {
		t.Errorf("expected latest duration, got: %v", metric.Duration)
	}
	if metric.RowsProcessed != 2000 {
		t.Errorf("expected latest rows, got: %d", metric.RowsProcessed)
	}
}

