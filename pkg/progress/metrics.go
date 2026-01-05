package progress

import (
	"fmt"
	"sync"
	"time"
)

// Metrics tracks operation performance metrics.
type Metrics struct {
	Operations map[string]*OperationMetrics
	mu         sync.RWMutex
}

// OperationMetrics holds timing and throughput information for a single operation.
type OperationMetrics struct {
	Name          string
	Duration      time.Duration
	RowsProcessed int64
	Throughput    float64 // rows/sec
	StartTime     time.Time
	EndTime       time.Time
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		Operations: make(map[string]*OperationMetrics),
	}
}

// Record adds or updates an operation metric.
func (m *Metrics) Record(name string, duration time.Duration, rows int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	throughput := float64(0)
	if duration.Seconds() > 0 {
		throughput = float64(rows) / duration.Seconds()
	}

	m.Operations[name] = &OperationMetrics{
		Name:          name,
		Duration:      duration,
		RowsProcessed: rows,
		Throughput:    throughput,
	}
}

// Get returns a copy of the operation metrics for the given name.
func (m *Metrics) Get(name string) *OperationMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metric, ok := m.Operations[name]; ok {
		copy := *metric
		return &copy
	}
	return nil
}

// Summary returns a formatted summary of all collected metrics.
func (m *Metrics) Summary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.Operations) == 0 {
		return ""
	}

	var result string
	result += "\nPerformance Metrics:\n"
	result += fmt.Sprintf("%-30s %15s %15s %20s\n", "Operation", "Duration", "Rows", "Throughput (rows/s)")
	result += fmt.Sprintf("%-30s %15s %15s %20s\n", "─────────", "────────", "────", "────────────────────")

	var totalDuration time.Duration
	var totalRows int64

	for _, metric := range m.Operations {
		result += fmt.Sprintf("%-30s %15s %15d %20.2f\n",
			truncateName(metric.Name, 30),
			metric.Duration.Round(time.Millisecond),
			metric.RowsProcessed,
			metric.Throughput)

		totalDuration += metric.Duration
		totalRows += metric.RowsProcessed
	}

	result += fmt.Sprintf("%-30s %15s %15s %20s\n", "─────────", "────────", "────", "────────────────────")

	avgThroughput := float64(0)
	if totalDuration.Seconds() > 0 {
		avgThroughput = float64(totalRows) / totalDuration.Seconds()
	}

	result += fmt.Sprintf("%-30s %15s %15d %20.2f\n",
		"TOTAL",
		totalDuration.Round(time.Millisecond),
		totalRows,
		avgThroughput)

	return result
}

// truncateName truncates a string to maxLen, adding "..." if needed.
func truncateName(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
