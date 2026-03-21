package checkpoint

import (
	"fmt"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/config"
	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

// ResumeOptions holds options for resuming from a checkpoint.
type ResumeOptions struct {
	// MaxAge is the maximum age of a checkpoint before it's considered expired.
	// If zero, defaults to 24 hours.
	MaxAge time.Duration

	// ValidateConfig ensures the config matches the checkpoint's config hash.
	ValidateConfig bool
}

// Validate validates a checkpoint state for resuming.
func Validate(state *State, cfg *config.Config, opts ResumeOptions) error {
	if state == nil {
		return fmt.Errorf("checkpoint state is nil")
	}

	// Check expiration
	if state.IsExpired(opts.MaxAge) {
		expiredErr := fmt.Errorf("checkpoint is %v old", time.Since(state.LastUpdated))
		return errors.Wrap(expiredErr, errors.ErrCheckpointExpired, "")
	}

	// Validate config hash if requested
	if opts.ValidateConfig {
		if err := state.ValidateConfigHash(cfg); err != nil {
			return errors.Wrap(err, errors.ErrCheckpointInvalid, "")
		}
	}

	return nil
}

// ResumeInfo provides information about a checkpoint that can be resumed.
type ResumeInfo struct {
	Operation   OperationType `json:"operation"`
	CreatedAt   time.Time     `json:"created_at"`
	LastUpdated time.Time     `json:"last_updated"`
	OutputDir   string        `json:"output_dir"`
}

// GetResumeInfo extracts resume information from a checkpoint state.
func GetResumeInfo(state *State) *ResumeInfo {
	if state == nil {
		return nil
	}

	return &ResumeInfo{
		Operation:   state.Operation,
		CreatedAt:   state.CreatedAt,
		LastUpdated: state.LastUpdated,
		OutputDir:   state.OutputDir,
	}
}
