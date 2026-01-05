package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

func TestBar_BasicOperations(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if bar == nil {
		t.Fatal("expected non-nil bar")
	}

	// Test basic operations
	if err := bar.Add(10); err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if err := bar.Set(50); err != nil {
		t.Errorf("Set failed: %v", err)
	}

	bar.Describe("new description")

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}
}

func TestBar_Spinner(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartSpinner(ctx, "test")

	if bar == nil {
		t.Fatal("expected non-nil spinner")
	}

	// Spinner should accept Add calls
	if err := bar.Add(1); err != nil {
		t.Errorf("Add failed: %v", err)
	}

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}
}

func TestBar_Add(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	// Add multiple times
	for i := 0; i < 10; i++ {
		if err := bar.Add(1); err != nil {
			t.Errorf("Add failed: %v", err)
		}
	}

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}
}

func TestBar_Set(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if err := bar.Set(0); err != nil {
		t.Errorf("Set(0) failed: %v", err)
	}

	if err := bar.Set(50); err != nil {
		t.Errorf("Set(50) failed: %v", err)
	}

	if err := bar.Set(100); err != nil {
		t.Errorf("Set(100) failed: %v", err)
	}

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}
}

func TestBar_Describe(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	bar.Describe("initial description")
	bar.Describe("updated description")

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}
}

func TestBar_Finish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish failed: %v", err)
	}

	// Finish should be idempotent
	if err := bar.Finish(); err != nil {
		t.Errorf("Second Finish failed: %v", err)
	}
}

func TestBar_Throughput(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	// Wait a bit to ensure elapsed time > 0
	time.Sleep(10 * time.Millisecond)

	throughput := bar.Throughput()
	if throughput < 0 {
		t.Errorf("expected non-negative throughput, got: %f", throughput)
	}

	// Add some progress
	_ = bar.Add(10)
	time.Sleep(10 * time.Millisecond)

	newThroughput := bar.Throughput()
	if newThroughput <= throughput {
		t.Errorf("expected throughput to increase, got: %f -> %f", throughput, newThroughput)
	}

	_ = bar.Finish()
}

func TestBar_Throughput_ZeroElapsed(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	// Throughput should handle zero elapsed time gracefully
	throughput := bar.Throughput()
	if throughput != 0 {
		t.Errorf("expected zero throughput for zero elapsed time, got: %f", throughput)
	}

	_ = bar.Finish()
}

func TestBar_Duration(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	initialDuration := bar.Duration()
	if initialDuration < 0 {
		t.Errorf("expected non-negative duration, got: %v", initialDuration)
	}

	time.Sleep(10 * time.Millisecond)
	updatedDuration := bar.Duration()
	if updatedDuration <= initialDuration {
		t.Errorf("expected duration to increase, got: %v -> %v", initialDuration, updatedDuration)
	}

	_ = bar.Finish()
}

func TestBar_IsComplete(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if bar.IsComplete() {
		t.Error("bar should not be complete initially")
	}

	_ = bar.Finish()
	if !bar.IsComplete() {
		t.Error("bar should be complete after Finish")
	}
}

func TestBar_String(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	str := bar.String()
	if !strings.Contains(str, "test") {
		t.Errorf("expected bar name in string, got: %s", str)
	}
	if !strings.Contains(str, "rows") {
		t.Errorf("expected 'rows' in string, got: %s", str)
	}

	_ = bar.Add(50)
	str = bar.String()
	if !strings.Contains(str, "50") {
		t.Errorf("expected progress in string, got: %s", str)
	}

	_ = bar.Finish()
}

func TestBar_String_Spinner(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartSpinner(ctx, "test")

	str := bar.String()
	if !strings.Contains(str, "test") {
		t.Errorf("expected spinner name in string, got: %s", str)
	}
	if !strings.Contains(str, "rows") {
		t.Errorf("expected 'rows' in string, got: %s", str)
	}

	_ = bar.Add(10)
	str = bar.String()
	if !strings.Contains(str, "10") {
		t.Errorf("expected progress in string, got: %s", str)
	}

	_ = bar.Finish()
}

func TestBar_Add_AfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish()

	// Add after finish should be no-op
	if err := bar.Add(10); err != nil {
		t.Errorf("Add after finish should succeed (no-op), got: %v", err)
	}
}

func TestBar_Set_AfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish()

	// Set after finish should be no-op
	if err := bar.Set(50); err != nil {
		t.Errorf("Set after finish should succeed (no-op), got: %v", err)
	}
}

func TestBar_Describe_AfterFinish(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	_ = bar.Finish()

	// Describe after finish should be no-op (no panic)
	bar.Describe("after finish")
}

func TestBar_Disabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: false,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	// All operations should handle nil pb gracefully
	if err := bar.Add(10); err != nil {
		t.Errorf("Add on disabled bar should succeed, got: %v", err)
	}

	if err := bar.Set(50); err != nil {
		t.Errorf("Set on disabled bar should succeed, got: %v", err)
	}

	bar.Describe("test") // Should not panic

	if err := bar.Finish(); err != nil {
		t.Errorf("Finish on disabled bar should succeed, got: %v", err)
	}

	// Throughput should still work
	throughput := bar.Throughput()
	if throughput < 0 {
		t.Errorf("expected non-negative throughput, got: %f", throughput)
	}
}

