package progress

import (
	"context"
	"io"
	"os"
	"sync"
	"time"
)

// Manager handles multiple concurrent progress bars and metrics collection.
type Manager struct {
	bars    map[string]*Bar
	mu      sync.Mutex
	enabled bool
	output  io.Writer
	metrics *Metrics
}

// Config holds configuration for creating a progress manager.
type Config struct {
	// Enabled controls whether progress bars are shown.
	// Set to false in CI/CD or verbose mode to avoid conflicts.
	Enabled bool

	// Output is the writer for progress bars (typically os.Stderr).
	Output io.Writer

	// ShowMetrics enables collection of performance metrics.
	ShowMetrics bool
}

// managerKeyType is a private type for context keys.
type managerKeyType struct{}

// managerKey is the context key for storing Manager instances.
var managerKey = managerKeyType{}

// NewManager creates a new progress bar manager with the given configuration.
func NewManager(cfg Config) *Manager {
	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	var metrics *Metrics
	if cfg.ShowMetrics {
		metrics = NewMetrics()
	}

	return &Manager{
		bars:    make(map[string]*Bar),
		enabled: cfg.Enabled,
		output:  cfg.Output,
		metrics: metrics,
	}
}

// StartBar creates and starts a new progress bar with a known total.
// If progress bars are disabled, this still tracks metrics but doesn't display anything.
func (m *Manager) StartBar(ctx context.Context, name string, total int64) *Bar {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bar *Bar
	if m.enabled {
		bar = newBar(name, total, m.output)
	} else {
		// Create a no-op bar that still tracks metrics
		bar = &Bar{
			name:    name,
			total:   total,
			started: timeNow(),
		}
	}

	m.bars[name] = bar
	return bar
}

// StartSpinner creates a spinner for operations with unknown totals.
func (m *Manager) StartSpinner(ctx context.Context, name string) *Bar {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bar *Bar
	if m.enabled {
		bar = newSpinner(name, m.output)
	} else {
		// Create a no-op spinner that still tracks metrics
		bar = &Bar{
			name:    name,
			total:   -1,
			started: timeNow(),
		}
	}

	m.bars[name] = bar
	return bar
}

// GetBar retrieves a progress bar by name.
func (m *Manager) GetBar(name string) *Bar {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.bars[name]
}

// Finish completes all active progress bars and collects final metrics.
func (m *Manager) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, bar := range m.bars {
		if !bar.IsComplete() {
			bar.Finish()
		}

		// Record metrics
		if m.metrics != nil {
			m.metrics.Record(name, bar.Duration(), bar.current)
		}
	}
}

// GetMetrics returns the collected metrics.
// Returns nil if metrics collection is not enabled.
func (m *Manager) GetMetrics() *Metrics {
	return m.metrics
}

// IsEnabled returns true if progress bars are enabled.
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// ToContext adds the progress manager to the given context.
func ToContext(ctx context.Context, m *Manager) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, managerKey, m)
}

// FromContext retrieves the progress manager from the given context.
// Returns nil if no manager is found.
func FromContext(ctx context.Context) *Manager {
	if ctx == nil {
		return nil
	}

	if m, ok := ctx.Value(managerKey).(*Manager); ok {
		return m
	}

	return nil
}

// timeNow is a variable for testing (can be mocked in tests).
var timeNow = func() time.Time {
	return time.Now()
}
