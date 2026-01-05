package progress

import (
	"fmt"
	"runtime"
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
	MemoryMB      float64 // Peak memory usage in MB
	QueryCount    int64   // Number of queries executed
}

// NewMetrics creates a new metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		Operations: make(map[string]*OperationMetrics),
	}
}

// Record adds or updates an operation metric.
func (m *Metrics) Record(name string, duration time.Duration, rows int64) {
	m.RecordWithDetails(name, duration, rows, 0, 0)
}

// RecordWithDetails adds or updates an operation metric with additional details.
func (m *Metrics) RecordWithDetails(name string, duration time.Duration, rows int64, queryCount int64, memoryMB float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	throughput := float64(0)
	if duration.Seconds() > 0 {
		throughput = float64(rows) / duration.Seconds()
	}

	// Get current memory usage if not provided
	if memoryMB == 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		memoryMB = float64(m.Alloc) / 1024 / 1024 // Convert bytes to MB
	}

	m.Operations[name] = &OperationMetrics{
		Name:          name,
		Duration:      duration,
		RowsProcessed: rows,
		Throughput:    throughput,
		MemoryMB:      memoryMB,
		QueryCount:    queryCount,
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
	result += fmt.Sprintf("%-30s %15s %15s %20s %12s %12s\n", "Operation", "Duration", "Rows", "Throughput (rows/s)", "Memory (MB)", "Queries")
	result += fmt.Sprintf("%-30s %15s %15s %20s %12s %12s\n", "─────────", "────────", "────", "────────────────────", "───────────", "───────")

	var totalDuration time.Duration
	var totalRows int64

	var totalMemoryMB float64
	var totalQueries int64

	for _, metric := range m.Operations {
		result += fmt.Sprintf("%-30s %15s %15d %20.2f %12.2f %12d\n",
			truncateName(metric.Name, 30),
			metric.Duration.Round(time.Millisecond),
			metric.RowsProcessed,
			metric.Throughput,
			metric.MemoryMB,
			metric.QueryCount)

		totalDuration += metric.Duration
		totalRows += metric.RowsProcessed
		totalMemoryMB += metric.MemoryMB
		totalQueries += metric.QueryCount
	}

	result += fmt.Sprintf("%-30s %15s %15s %20s %12s %12s\n", "─────────", "────────", "────", "────────────────────", "───────────", "───────")

	avgThroughput := float64(0)
	if totalDuration.Seconds() > 0 {
		avgThroughput = float64(totalRows) / totalDuration.Seconds()
	}

	result += fmt.Sprintf("%-30s %15s %15d %20.2f %12.2f %12d\n",
		"TOTAL",
		totalDuration.Round(time.Millisecond),
		totalRows,
		avgThroughput,
		totalMemoryMB,
		totalQueries)

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
