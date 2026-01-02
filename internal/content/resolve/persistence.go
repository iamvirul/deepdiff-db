// Package resolve provides the core resolution engine for applying conflict
// resolution strategies to detected conflicts between production and development databases.
package resolve

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
)

// ResolutionFile represents the JSON structure for saved resolutions.
type ResolutionFile struct {
	// Version is the file format version.
	Version string `json:"version"`
	// CreatedAt is when the file was first created.
	CreatedAt string `json:"created_at"`
	// UpdatedAt is when the file was last updated.
	UpdatedAt string `json:"updated_at"`
	// Resolutions is the list of conflict resolutions.
	Resolutions []Resolution `json:"resolutions"`
}

const resolutionFileVersion = "1.0"

// SaveResolutions writes resolutions to a JSON file.
// If the file exists, it updates the UpdatedAt timestamp.
// If the file doesn't exist, it creates a new file with both timestamps.
func SaveResolutions(resolutions []Resolution, filePath string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	// Try to load existing file to preserve CreatedAt
	var createdAt string
	if existing, err := LoadResolutions(filePath); err == nil {
		createdAt = existing.CreatedAt
	} else {
		createdAt = now
	}

	file := ResolutionFile{
		Version:     resolutionFileVersion,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
		Resolutions: resolutions,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal resolutions: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("write resolutions file: %w", err)
	}

	return nil
}

// LoadResolutions reads resolutions from a JSON file.
// Returns an error if the file doesn't exist or is malformed.
func LoadResolutions(filePath string) (*ResolutionFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read resolutions file: %w", err)
	}

	var file ResolutionFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse resolutions file: %w", err)
	}

	return &file, nil
}

// MergeResolutions merges saved resolutions with new conflicts.
// For conflicts that exist in both, the saved resolution is preserved.
// For new conflicts (not in saved), they are added with pending status.
// For saved resolutions whose conflicts no longer exist, they are removed.
func MergeResolutions(saved []Resolution, conflicts content.Conflicts) []Resolution {
	// Build a map of saved resolutions by conflict key
	savedMap := make(map[string]Resolution)
	for _, res := range saved {
		key := makeConflictKey(res.Conflict)
		savedMap[key] = res
	}

	// Build result: for each current conflict, use saved resolution or create pending
	merged := make([]Resolution, 0, len(conflicts.Conflicts))
	for _, conflict := range conflicts.Conflicts {
		key := makeConflictKey(conflict)
		if savedRes, exists := savedMap[key]; exists {
			// Update the conflict data (hashes might have changed) but keep the decision
			savedRes.Conflict = conflict
			merged = append(merged, savedRes)
		} else {
			// New conflict, create pending resolution
			merged = append(merged, Resolution{
				Conflict: conflict,
				Strategy: StrategyManual,
				Decision: DecisionPending,
				Resolved: false,
			})
		}
	}

	return merged
}

// UpdateResolution updates a single resolution in the list.
// Returns the updated list with the resolution at the specified index modified.
func UpdateResolution(resolutions []Resolution, index int, strategy Strategy, decision Decision) []Resolution {
	if index < 0 || index >= len(resolutions) {
		return resolutions
	}

	resolutions[index].Strategy = strategy
	resolutions[index].Decision = decision
	resolutions[index].Resolved = decision != DecisionPending

	return resolutions
}

// ApplyBulkResolution applies the same resolution to multiple conflicts.
// It updates all resolutions matching the filter criteria.
func ApplyBulkResolution(resolutions []Resolution, table string, allTables bool, strategy Strategy) []Resolution {
	decision := strategyToDecision(strategy)

	for i := range resolutions {
		// Skip already resolved
		if resolutions[i].Resolved {
			continue
		}

		// Check if this resolution matches the filter
		if allTables || resolutions[i].Conflict.Table == table {
			resolutions[i].Strategy = strategy
			resolutions[i].Decision = decision
			resolutions[i].Resolved = decision != DecisionPending
		}
	}

	return resolutions
}

// strategyToDecision converts a strategy to its corresponding decision.
func strategyToDecision(strategy Strategy) Decision {
	switch strategy {
	case StrategyOurs:
		return DecisionKeepProd
	case StrategyTheirs:
		return DecisionUseDev
	default:
		return DecisionPending
	}
}

// GetPendingResolutions returns only resolutions that are still pending.
func GetPendingResolutions(resolutions []Resolution) []Resolution {
	pending := make([]Resolution, 0)
	for _, res := range resolutions {
		if !res.Resolved {
			pending = append(pending, res)
		}
	}
	return pending
}

// GetPendingCount returns the count of pending resolutions.
func GetPendingCount(resolutions []Resolution) int {
	count := 0
	for _, res := range resolutions {
		if !res.Resolved {
			count++
		}
	}
	return count
}

// GroupByTable groups resolutions by table name.
func GroupByTable(resolutions []Resolution) map[string][]Resolution {
	grouped := make(map[string][]Resolution)
	for _, res := range resolutions {
		table := res.Conflict.Table
		grouped[table] = append(grouped[table], res)
	}
	return grouped
}

// GetTableOrder returns a sorted list of table names from resolutions.
func GetTableOrder(resolutions []Resolution) []string {
	seen := make(map[string]bool)
	order := make([]string, 0)

	for _, res := range resolutions {
		table := res.Conflict.Table
		if !seen[table] {
			seen[table] = true
			order = append(order, table)
		}
	}

	return order
}
