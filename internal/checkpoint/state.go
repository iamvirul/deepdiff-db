package checkpoint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iamvirul/deepdiff-db/pkg/config"
)

// OperationType represents the type of operation being checkpointed.
type OperationType string

const (
	// OperationTypeHashTable identifies a hash-table operation checkpoint.
	OperationTypeHashTable OperationType = "hash_table"
	// OperationTypeGeneratePack identifies a generate-pack operation checkpoint.
	OperationTypeGeneratePack OperationType = "generate_pack"
	// OperationTypeApplyPack identifies an apply-pack operation checkpoint.
	OperationTypeApplyPack OperationType = "apply_pack"
)

// State represents the checkpoint state that can be saved and resumed.
type State struct {
	// Metadata
	Version     string        `json:"version"`      // Checkpoint format version
	Operation   OperationType `json:"operation"`    // Type of operation
	CreatedAt   time.Time     `json:"created_at"`   // When checkpoint was created
	LastUpdated time.Time     `json:"last_updated"` // Last update time
	ConfigHash  string        `json:"config_hash"`  // Hash of config for validation
	OutputDir   string        `json:"output_dir"`   // Output directory path

	// HashTable state
	HashTableState *HashTableState `json:"hash_table_state,omitempty"`

	// GeneratePack state
	GeneratePackState *GeneratePackState `json:"generate_pack_state,omitempty"`

	// ApplyPack state
	ApplyPackState *ApplyPackState `json:"apply_pack_state,omitempty"`
}

// HashTableState tracks progress for table hashing operations.
type HashTableState struct {
	CompletedTables []string                     `json:"completed_tables"`  // Tables fully hashed
	CurrentTable    string                       `json:"current_table"`     // Table currently being processed
	CurrentRowCount int64                        `json:"current_row_count"` // Rows processed in current table
	Hashes          map[string]map[string]string `json:"hashes"`            // Completed hash results
}

// GeneratePackState tracks progress for pack generation.
type GeneratePackState struct {
	CompletedTables []string `json:"completed_tables"` // Tables fully processed
	CurrentTable    string   `json:"current_table"`    // Table currently being processed
	Statements      []string `json:"statements"`       // Generated statements so far
}

// ApplyPackState tracks progress for pack application.
type ApplyPackState struct {
	ExecutedStatements int    `json:"executed_statements"` // Number of statements executed
	TotalStatements    int    `json:"total_statements"`    // Total statements in pack
	PackPath           string `json:"pack_path"`           // Path to pack file
}

// CurrentVersion is the checkpoint format version.
const CurrentVersion = "1.0"

// ComputeConfigHash computes a SHA-256 hash of the configuration for validation.
func ComputeConfigHash(cfg *config.Config) (string, error) {
	// Serialize config to JSON for hashing
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// NewState creates a new checkpoint state for the given operation.
func NewState(operation OperationType, outputDir string, cfg *config.Config) (*State, error) {
	configHash, err := ComputeConfigHash(cfg)
	if err != nil {
		return nil, fmt.Errorf("compute config hash: %w", err)
	}

	now := time.Now()
	return &State{
		Version:     CurrentVersion,
		Operation:   operation,
		CreatedAt:   now,
		LastUpdated: now,
		ConfigHash:  configHash,
		OutputDir:   outputDir,
	}, nil
}

// ValidateConfigHash validates that the checkpoint's config hash matches the current config.
func (s *State) ValidateConfigHash(cfg *config.Config) error {
	expectedHash, err := ComputeConfigHash(cfg)
	if err != nil {
		return fmt.Errorf("compute config hash: %w", err)
	}

	if s.ConfigHash != expectedHash {
		return fmt.Errorf("config hash mismatch: checkpoint was created with different configuration")
	}

	return nil
}

// IsExpired checks if the checkpoint is too old to safely resume from.
// Default expiration: 24 hours.
func (s *State) IsExpired(maxAge time.Duration) bool {
	if maxAge == 0 {
		maxAge = 24 * time.Hour // Default: 24 hours
	}
	return time.Since(s.LastUpdated) > maxAge
}
