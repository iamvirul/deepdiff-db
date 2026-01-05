package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Logger wraps Go's standard slog.Logger with convenience methods and
// additional functionality like operation timing and metrics collection.
type Logger struct {
	slog    *slog.Logger
	metrics *MetricsCollector
	mu      sync.RWMutex
}

// Config holds configuration for creating a new Logger.
type Config struct {
	// Level is the minimum log level (Debug, Info, Warn, Error).
	// Default is Info.
	Level slog.Level

	// Format specifies the output format: "json" or "text".
	// Default is "text".
	Format string

	// Output is the primary output destination (typically os.Stdout).
	// Default is os.Stdout.
	Output io.Writer

	// FileOutput is an optional secondary output for writing logs to a file.
	// If nil, logs are only written to Output.
	FileOutput io.Writer

	// WithSource adds source code location to log entries when true.
	// Useful for debugging but adds overhead.
	WithSource bool

	// EnableMetrics enables collection of operation metrics when true.
	EnableMetrics bool
}

// MetricsCollector tracks operation metrics.
type MetricsCollector struct {
	operations map[string]*OperationMetric
	mu         sync.RWMutex
}

// OperationMetric holds timing and count information for an operation.
type OperationMetric struct {
	Name      string
	Count     int64
	TotalTime time.Duration
	LastTime  time.Duration
}

// New creates a new Logger with the given configuration.
// If config fields are zero values, sensible defaults are used.
func New(cfg Config) *Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	if cfg.Format == "" {
		cfg.Format = "text"
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.WithSource,
	}

	var handler slog.Handler

	// Create appropriate handler based on format
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	// If file output is specified, create a multi-writer handler
	if cfg.FileOutput != nil {
		multiWriter := io.MultiWriter(cfg.Output, cfg.FileOutput)
		switch cfg.Format {
		case "json":
			handler = slog.NewJSONHandler(multiWriter, opts)
		default:
			handler = slog.NewTextHandler(multiWriter, opts)
		}
	}

	l := &Logger{
		slog: slog.New(handler),
	}

	if cfg.EnableMetrics {
		l.metrics = &MetricsCollector{
			operations: make(map[string]*OperationMetric),
		}
	}

	return l
}

// Debug logs a debug-level message with optional key-value pairs.
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Info logs an info-level message with optional key-value pairs.
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning-level message with optional key-value pairs.
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs an error-level message with optional key-value pairs.
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// With returns a new Logger with the given key-value pairs added as context.
// These fields will be included in all log entries from the returned logger.
//
// Example:
//
//	tableLogger := logger.With("table", "users", "operation", "hash")
//	tableLogger.Info("processing rows") // Will include table and operation fields
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog:    l.slog.With(args...),
		metrics: l.metrics, // Share metrics collector
	}
}

// WithTable returns a new Logger with the table name added to all log entries.
func (l *Logger) WithTable(table string) *Logger {
	return l.With(FieldTable, table)
}

// WithOperation returns a new Logger with the operation name added to all log entries.
func (l *Logger) WithOperation(op string) *Logger {
	return l.With(FieldOperation, op)
}

// WithDatabase returns a new Logger with database information added to all log entries.
func (l *Logger) WithDatabase(driver, database string) *Logger {
	return l.With(FieldDriver, driver, FieldDatabase, database)
}

// LogOperation executes the given function while measuring its duration.
// It logs the operation start, completion, and any errors that occur.
// If metrics are enabled, it records timing information.
//
// Example:
//
//	err := logger.LogOperation(ctx, "load_schema", func() error {
//	    // ... perform operation
//	    return nil
//	})
func (l *Logger) LogOperation(ctx context.Context, name string, fn func() error) error {
	l.Debug("operation starting", FieldOperation, name)

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	if err != nil {
		l.Error("operation failed",
			FieldOperation, name,
			FieldDuration, duration.Milliseconds(),
			FieldError, err.Error())
	} else {
		l.Info("operation complete",
			FieldOperation, name,
			FieldDuration, duration.Milliseconds())
	}

	// Record metrics if enabled
	if l.metrics != nil {
		l.recordMetric(name, duration)
	}

	return err
}

// recordMetric records timing information for an operation.
func (l *Logger) recordMetric(name string, duration time.Duration) {
	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()

	metric, exists := l.metrics.operations[name]
	if !exists {
		metric = &OperationMetric{
			Name: name,
		}
		l.metrics.operations[name] = metric
	}

	metric.Count++
	metric.TotalTime += duration
	metric.LastTime = duration
}

// GetMetrics returns a copy of collected metrics.
// Returns nil if metrics are not enabled.
func (l *Logger) GetMetrics() map[string]OperationMetric {
	if l.metrics == nil {
		return nil
	}

	l.metrics.mu.RLock()
	defer l.metrics.mu.RUnlock()

	result := make(map[string]OperationMetric, len(l.metrics.operations))
	for name, metric := range l.metrics.operations {
		result[name] = *metric
	}

	return result
}

// PrintMetricsSummary outputs a formatted summary of all collected metrics.
func (l *Logger) PrintMetricsSummary(w io.Writer) {
	metrics := l.GetMetrics()
	if len(metrics) == 0 {
		return
	}

	fmt.Fprintf(w, "\nOperation Metrics:\n")
	fmt.Fprintf(w, "%-30s %10s %15s %15s\n", "Operation", "Count", "Total Time", "Avg Time")
	fmt.Fprintf(w, "%-30s %10s %15s %15s\n", "─────────", "─────", "──────────", "────────")

	for _, metric := range metrics {
		avgTime := time.Duration(0)
		if metric.Count > 0 {
			avgTime = metric.TotalTime / time.Duration(metric.Count)
		}

		fmt.Fprintf(w, "%-30s %10d %15s %15s\n",
			metric.Name,
			metric.Count,
			metric.TotalTime.Round(time.Millisecond),
			avgTime.Round(time.Millisecond))
	}
}

// ParseLevel converts a string log level to slog.Level.
// Returns slog.LevelInfo if the string is not recognized.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
