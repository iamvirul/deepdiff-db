package main

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

func TestNewManager_DefaultOutput(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  nil, // Should default to os.Stderr
	})

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewManager_WithMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: true,
	})

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}

	metrics := mgr.GetMetrics()
	if metrics == nil {
		t.Error("expected metrics to be enabled")
	}
}

func TestNewManager_WithoutMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: false,
	})

	metrics := mgr.GetMetrics()
	if metrics != nil {
		t.Error("expected metrics to be disabled")
	}
}

func TestManager_StartBar_Multiple(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()

	bar1 := mgr.StartBar(ctx, "bar1", 100)
	bar2 := mgr.StartBar(ctx, "bar2", 200)

	if bar1 == nil || bar2 == nil {
		t.Fatal("expected non-nil bars")
	}

	// Both should be retrievable
	retrieved1 := mgr.GetBar("bar1")
	retrieved2 := mgr.GetBar("bar2")

	if retrieved1 != bar1 {
		t.Error("expected same bar instance")
	}
	if retrieved2 != bar2 {
		t.Error("expected same bar instance")
	}
}

func TestManager_StartBar_Disabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: false,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)

	if bar == nil {
		t.Fatal("expected non-nil bar even when disabled")
	}

	// Bar should still track metrics
	if err := bar.Add(10); err != nil {
		t.Errorf("Add should work when disabled: %v", err)
	}
}

func TestManager_StartSpinner_Multiple(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()

	spinner1 := mgr.StartSpinner(ctx, "spinner1")
	spinner2 := mgr.StartSpinner(ctx, "spinner2")

	if spinner1 == nil || spinner2 == nil {
		t.Fatal("expected non-nil spinners")
	}
}

func TestManager_StartSpinner_Disabled(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: false,
	})

	ctx := context.Background()
	spinner := mgr.StartSpinner(ctx, "test")

	if spinner == nil {
		t.Fatal("expected non-nil spinner even when disabled")
	}

	// Spinner should still track metrics
	if err := spinner.Add(1); err != nil {
		t.Errorf("Add should work when disabled: %v", err)
	}
}

func TestManager_GetBar_NotFound(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	bar := mgr.GetBar("nonexistent")
	if bar != nil {
		t.Error("expected nil for nonexistent bar")
	}
}

func TestManager_Finish_WithMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: true,
	})

	ctx := context.Background()

	bar1 := mgr.StartBar(ctx, "bar1", 100)
	bar2 := mgr.StartBar(ctx, "bar2", 200)

	_ = bar1.Add(50)
	_ = bar2.Add(100)

	// Wait a bit for duration
	time.Sleep(10 * time.Millisecond)

	mgr.Finish()

	// Check metrics were recorded
	metrics := mgr.GetMetrics()
	if metrics == nil {
		t.Fatal("expected metrics")
	}

	metric1 := metrics.Get("bar1")
	if metric1 == nil {
		t.Error("expected bar1 metric")
	}

	metric2 := metrics.Get("bar2")
	if metric2 == nil {
		t.Error("expected bar2 metric")
	}
}

func TestManager_Finish_WithoutMetrics(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled:     true,
		Output:      io.Discard,
		ShowMetrics: false,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)
	_ = bar.Add(50)

	mgr.Finish()

	// Should not panic even without metrics
	metrics := mgr.GetMetrics()
	if metrics != nil {
		t.Error("expected no metrics")
	}
}

func TestManager_Finish_AlreadyComplete(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	bar := mgr.StartBar(ctx, "test", 100)
	_ = bar.Finish()

	// Finish should handle already-completed bars
	mgr.Finish()
}

func TestToContext_FromContext(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()
	ctx = progress.ToContext(ctx, mgr)

	retrieved := progress.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected non-nil manager from context")
	}

	if retrieved.IsEnabled() != mgr.IsEnabled() {
		t.Error("expected same enabled state")
	}
}

func TestToContext_NilContext(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := progress.ToContext(context.TODO(), mgr)
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	retrieved := progress.FromContext(ctx)
	if retrieved == nil {
		t.Fatal("expected manager from context")
	}
}

func TestFromContext_NilContext(t *testing.T) {
	mgr := progress.FromContext(context.TODO())
	if mgr != nil {
		t.Error("expected nil manager from context without manager")
	}
}

func TestFromContext_WrongType(t *testing.T) {
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")
	mgr := progress.FromContext(ctx)
	if mgr != nil {
		t.Error("expected nil manager for wrong type in context")
	}
}

func TestManager_ConcurrentBars(t *testing.T) {
	mgr := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	ctx := context.Background()

	// Create multiple bars concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			bar := mgr.StartBar(ctx, "bar", 100)
			_ = bar.Add(10)
			_ = bar.Finish()
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Finish should handle all bars
	mgr.Finish()
}

func TestManager_IsEnabled(t *testing.T) {
	mgr1 := progress.NewManager(progress.Config{
		Enabled: true,
		Output:  io.Discard,
	})

	if !mgr1.IsEnabled() {
		t.Error("expected enabled")
	}

	mgr2 := progress.NewManager(progress.Config{
		Enabled: false,
	})

	if mgr2.IsEnabled() {
		t.Error("expected disabled")
	}
}
