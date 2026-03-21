package checkpoint

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

// CheckpointFileName is the name of the checkpoint file.
const CheckpointFileName = ".deepdiffdb_checkpoint.json"

// Manager handles checkpoint save/load operations.
type Manager struct {
	checkpointPath string
	state          *State
}

// NewManager creates a new checkpoint manager for the given output directory.
func NewManager(outputDir string) *Manager {
	checkpointPath := filepath.Join(outputDir, CheckpointFileName)
	return &Manager{
		checkpointPath: checkpointPath,
	}
}

// Save writes the current state to the checkpoint file.
func (m *Manager) Save(state *State) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	state.LastUpdated = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, errors.ErrCheckpointWrite, "marshal checkpoint state")
	}

	// Ensure output directory exists
	dir := filepath.Dir(m.checkpointPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return errors.Wrap(err, errors.ErrCheckpointWrite, "create checkpoint directory")
	}

	// Write checkpoint file atomically (write to temp, then rename)
	tmpPath := m.checkpointPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return errors.Wrap(err, errors.ErrCheckpointWrite, "write checkpoint file")
	}

	if err := os.Rename(tmpPath, m.checkpointPath); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file
		return errors.Wrap(err, errors.ErrCheckpointWrite, "rename checkpoint file")
	}

	m.state = state
	return nil
}

// Load reads the checkpoint state from the checkpoint file.
func (m *Manager) Load() (*State, error) {
	data, err := os.ReadFile(m.checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists - not an error
		}
		return nil, errors.Wrap(err, errors.ErrCheckpointRead, "read checkpoint file")
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, errors.Wrap(err, errors.ErrCheckpointInvalid, "parse checkpoint file")
	}

	// Validate version
	if state.Version != CurrentVersion {
		versionErr := fmt.Errorf("unsupported checkpoint version: %s (expected %s)", state.Version, CurrentVersion)
		return nil, errors.Wrap(versionErr, errors.ErrCheckpointInvalid, "")
	}

	m.state = &state
	return &state, nil
}

// Update updates the checkpoint state and saves it.
func (m *Manager) Update(updateFn func(*State) error) error {
	state := m.state
	if state == nil {
		return fmt.Errorf("no checkpoint state loaded")
	}

	if err := updateFn(state); err != nil {
		return err
	}

	return m.Save(state)
}

// Delete removes the checkpoint file.
func (m *Manager) Delete() error {
	if _, err := os.Stat(m.checkpointPath); os.IsNotExist(err) {
		return nil // Already deleted - not an error
	}

	if err := os.Remove(m.checkpointPath); err != nil {
		return errors.Wrap(err, errors.ErrCheckpointWrite, "delete checkpoint file")
	}

	m.state = nil
	return nil
}

// Path returns the checkpoint file path.
func (m *Manager) Path() string {
	return m.checkpointPath
}

// HasCheckpoint checks if a checkpoint file exists.
func (m *Manager) HasCheckpoint() bool {
	_, err := os.Stat(m.checkpointPath)
	return err == nil
}

// managerKeyType is a private type for context keys.
type managerKeyType struct{}

// managerKey is the context key for storing Manager instances.
var managerKey = managerKeyType{}

// ToContext adds the checkpoint manager to the given context.
func ToContext(ctx context.Context, m *Manager) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, managerKey, m)
}

// FromContext retrieves the checkpoint manager from the given context.
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
