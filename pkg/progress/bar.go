package progress

import (
	"fmt"
	"io"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Bar wraps progressbar.ProgressBar with additional functionality.
type Bar struct {
	pb        *progressbar.ProgressBar
	name      string
	started   time.Time
	total     int64
	current   int64
	completed bool
}

// newBar creates a new progress bar.
func newBar(name string, total int64, output io.Writer) *Bar {
	pb := progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription(name),
		progressbar.OptionSetWriter(output),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("rows"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond), // Update max every 100ms
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	return &Bar{
		pb:      pb,
		name:    name,
		started: time.Now(),
		total:   total,
	}
}

// newSpinner creates a spinner for unknown totals.
func newSpinner(name string, output io.Writer) *Bar {
	pb := progressbar.NewOptions64(
		-1, // -1 indicates unknown total (spinner mode)
		progressbar.OptionSetDescription(name),
		progressbar.OptionSetWriter(output),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionClearOnFinish(),
	)

	return &Bar{
		pb:      pb,
		name:    name,
		started: time.Now(),
		total:   -1, // Spinner mode
	}
}

// Add increments the progress bar by n.
func (b *Bar) Add(n int) error {
	if b.completed {
		return nil
	}
	b.current += int64(n)
	return b.pb.Add(n)
}

// Set sets the absolute progress value.
func (b *Bar) Set(n int64) error {
	if b.completed {
		return nil
	}
	b.current = n
	return b.pb.Set64(n)
}

// Describe updates the progress bar description.
func (b *Bar) Describe(desc string) {
	if b.completed {
		return
	}
	b.pb.Describe(desc)
}

// Finish completes the progress bar.
func (b *Bar) Finish() error {
	if b.completed {
		return nil
	}
	b.completed = true
	return b.pb.Finish()
}

// Throughput returns the current throughput in rows/second.
func (b *Bar) Throughput() float64 {
	elapsed := time.Since(b.started).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(b.current) / elapsed
}

// Duration returns the elapsed time since the bar started.
func (b *Bar) Duration() time.Duration {
	return time.Since(b.started)
}

// IsComplete returns true if the bar has been finished.
func (b *Bar) IsComplete() bool {
	return b.completed
}

// String returns a string representation of the bar's current state.
func (b *Bar) String() string {
	if b.total < 0 {
		return fmt.Sprintf("%s: %d rows (%.2f rows/s)",
			b.name, b.current, b.Throughput())
	}
	percent := float64(0)
	if b.total > 0 {
		percent = float64(b.current) / float64(b.total) * 100
	}
	return fmt.Sprintf("%s: %d/%d rows (%.1f%%, %.2f rows/s)",
		b.name, b.current, b.total, percent, b.Throughput())
}
