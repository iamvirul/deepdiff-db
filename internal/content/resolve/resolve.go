// Package resolve provides the core resolution engine for applying conflict
// resolution strategies to detected conflicts between production and development databases.
package resolve

import (
	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

// Strategy represents a conflict resolution strategy.
type Strategy string

const (
	// StrategyOurs keeps the production version.
	StrategyOurs Strategy = "ours"
	// StrategyTheirs uses the development version.
	StrategyTheirs Strategy = "theirs"
	// StrategyManual requires manual review.
	StrategyManual Strategy = "manual"
)

// Decision represents the resolution decision for a conflict.
type Decision string

const (
	// DecisionKeepProd indicates the production version should be kept.
	DecisionKeepProd Decision = "keep_prod"
	// DecisionUseDev indicates the development version should be used.
	DecisionUseDev Decision = "use_dev"
	// DecisionPending indicates the conflict requires manual review.
	DecisionPending Decision = "pending"
)

// Resolution represents the result of applying a resolution strategy to a conflict.
type Resolution struct {
	// Conflict is the original conflict being resolved.
	Conflict content.Conflict `json:"conflict"`
	// Strategy is the strategy that was applied.
	Strategy Strategy `json:"strategy"`
	// Decision is the resolution decision.
	Decision Decision `json:"decision"`
	// Resolved indicates whether the conflict was automatically resolved.
	Resolved bool `json:"resolved"`
}

// GetStrategyForTable returns the resolution strategy for the given table.
// It checks the config for a table-specific override, falls back to the default
// strategy, and returns StrategyManual if no config is provided.
func GetStrategyForTable(table string, cfg *config.Config) Strategy {
	if cfg == nil {
		return StrategyManual
	}

	strategyStr := cfg.ConflictResolution.GetStrategyForTable(table)
	return parseStrategy(strategyStr)
}

// parseStrategy converts a string strategy to the Strategy type.
func parseStrategy(s string) Strategy {
	switch s {
	case config.StrategyOurs:
		return StrategyOurs
	case config.StrategyTheirs:
		return StrategyTheirs
	case config.StrategyManual:
		return StrategyManual
	default:
		return StrategyManual
	}
}

// ApplyStrategy applies the given strategy to a conflict and returns a Resolution.
func ApplyStrategy(conflict content.Conflict, strategy Strategy) Resolution {
	res := Resolution{
		Conflict: conflict,
		Strategy: strategy,
	}

	switch strategy {
	case StrategyOurs:
		res.Decision = DecisionKeepProd
		res.Resolved = true
	case StrategyTheirs:
		res.Decision = DecisionUseDev
		res.Resolved = true
	case StrategyManual:
		res.Decision = DecisionPending
		res.Resolved = false
	default:
		res.Decision = DecisionPending
		res.Resolved = false
	}

	return res
}

// Conflicts applies resolution strategies to all conflicts based on the config.
// It looks up the appropriate strategy for each conflict's table and applies it.
func Conflicts(conflicts content.Conflicts, cfg *config.Config) []Resolution {
	resolutions := make([]Resolution, 0, len(conflicts.Conflicts))

	for _, conflict := range conflicts.Conflicts {
		strategy := GetStrategyForTable(conflict.Table, cfg)
		resolution := ApplyStrategy(conflict, strategy)
		resolutions = append(resolutions, resolution)
	}

	return resolutions
}

// FilterResolved returns a new Conflicts containing only conflicts that were not resolved.
// This is useful for identifying conflicts that still require manual intervention.
func FilterResolved(conflicts content.Conflicts, resolutions []Resolution) content.Conflicts {
	// Build a map of resolved conflict keys for quick lookup
	resolvedKeys := make(map[string]bool)
	for _, res := range resolutions {
		if res.Resolved {
			key := makeConflictKey(res.Conflict)
			resolvedKeys[key] = true
		}
	}

	// Filter out resolved conflicts
	unresolved := content.Conflicts{}
	for _, conflict := range conflicts.Conflicts {
		key := makeConflictKey(conflict)
		if !resolvedKeys[key] {
			unresolved.Conflicts = append(unresolved.Conflicts, conflict)
		}
	}

	return unresolved
}

// FilterUnresolved returns a new Conflicts containing only conflicts that were resolved.
// This is useful for identifying conflicts that have been automatically handled.
func FilterUnresolved(conflicts content.Conflicts, resolutions []Resolution) content.Conflicts {
	// Build a map of resolved conflict keys for quick lookup
	resolvedKeys := make(map[string]bool)
	for _, res := range resolutions {
		if res.Resolved {
			key := makeConflictKey(res.Conflict)
			resolvedKeys[key] = true
		}
	}

	// Keep only resolved conflicts
	resolved := content.Conflicts{}
	for _, conflict := range conflicts.Conflicts {
		key := makeConflictKey(conflict)
		if resolvedKeys[key] {
			resolved.Conflicts = append(resolved.Conflicts, conflict)
		}
	}

	return resolved
}

// makeConflictKey creates a unique key for a conflict based on table and row key.
func makeConflictKey(conflict content.Conflict) string {
	return conflict.Table + ":" + conflict.Key
}

// CountByDecision returns a map of decision counts from the resolutions.
func CountByDecision(resolutions []Resolution) map[Decision]int {
	counts := make(map[Decision]int)
	for _, res := range resolutions {
		counts[res.Decision]++
	}
	return counts
}

// CountByStrategy returns a map of strategy counts from the resolutions.
func CountByStrategy(resolutions []Resolution) map[Strategy]int {
	counts := make(map[Strategy]int)
	for _, res := range resolutions {
		counts[res.Strategy]++
	}
	return counts
}

// GetResolutionsForTable filters resolutions to only include those for the specified table.
func GetResolutionsForTable(resolutions []Resolution, table string) []Resolution {
	filtered := make([]Resolution, 0)
	for _, res := range resolutions {
		if res.Conflict.Table == table {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

// GetResolvedResolutions filters resolutions to only include those that were resolved.
func GetResolvedResolutions(resolutions []Resolution) []Resolution {
	resolved := make([]Resolution, 0)
	for _, res := range resolutions {
		if res.Resolved {
			resolved = append(resolved, res)
		}
	}
	return resolved
}

// GetUnresolvedResolutions filters resolutions to only include those that require manual review.
func GetUnresolvedResolutions(resolutions []Resolution) []Resolution {
	unresolved := make([]Resolution, 0)
	for _, res := range resolutions {
		if !res.Resolved {
			unresolved = append(unresolved, res)
		}
	}
	return unresolved
}

// FilterDataDiffByResolutions filters the Updated keys in a DataDiff based on resolutions.
// Keys resolved with "theirs" (use_dev) remain in the Updated list.
// Keys resolved with "ours" (keep_prod) or "manual" (pending) are removed.
// Added and Removed keys are not affected by conflict resolution.
// Returns the filtered DataDiff and counts of excluded keys by decision.
func FilterDataDiffByResolutions(diff content.DataDiff, resolutions []Resolution) (content.DataDiff, map[Decision]int) {
	// Build lookup maps for quick access
	// Map: table -> key -> decision
	decisionMap := make(map[string]map[string]Decision)
	for _, res := range resolutions {
		if decisionMap[res.Conflict.Table] == nil {
			decisionMap[res.Conflict.Table] = make(map[string]Decision)
		}
		decisionMap[res.Conflict.Table][res.Conflict.Key] = res.Decision
	}

	excludedCounts := make(map[Decision]int)
	filteredTables := make([]content.TableDataDiff, 0, len(diff.Tables))

	for _, td := range diff.Tables {
		filteredTd := content.TableDataDiff{
			Table:   td.Table,
			Added:   td.Added,   // Added keys are not affected
			Removed: td.Removed, // Removed keys are not affected
		}

		// Filter Updated keys based on resolutions
		tableDecisions := decisionMap[td.Table]
		for _, key := range td.Updated {
			decision, hasResolution := tableDecisions[key]
			if !hasResolution {
				// No resolution for this key, include it (backward compatible)
				filteredTd.Updated = append(filteredTd.Updated, key)
				continue
			}

			switch decision {
			case DecisionUseDev:
				// "theirs" strategy: include in pack (update prod with dev)
				filteredTd.Updated = append(filteredTd.Updated, key)
			case DecisionKeepProd:
				// "ours" strategy: exclude from pack (keep prod value)
				excludedCounts[DecisionKeepProd]++
			case DecisionPending:
				// "manual" strategy: exclude from pack (needs review)
				excludedCounts[DecisionPending]++
			default:
				// Unknown decision, include by default
				filteredTd.Updated = append(filteredTd.Updated, key)
			}
		}

		filteredTables = append(filteredTables, filteredTd)
	}

	return content.DataDiff{Tables: filteredTables}, excludedCounts
}

// ResolutionSummary contains summary statistics about conflict resolution.
type ResolutionSummary struct {
	TotalConflicts  int              `json:"total_conflicts"`
	ResolvedCount   int              `json:"resolved_count"`
	UnresolvedCount int              `json:"unresolved_count"`
	ByStrategy      map[Strategy]int `json:"by_strategy"`
	ByDecision      map[Decision]int `json:"by_decision"`
	ByTable         map[string]int   `json:"by_table"`
}

// BuildResolutionSummary creates a summary of resolution statistics.
func BuildResolutionSummary(resolutions []Resolution) ResolutionSummary {
	summary := ResolutionSummary{
		TotalConflicts: len(resolutions),
		ByStrategy:     make(map[Strategy]int),
		ByDecision:     make(map[Decision]int),
		ByTable:        make(map[string]int),
	}

	for _, res := range resolutions {
		summary.ByStrategy[res.Strategy]++
		summary.ByDecision[res.Decision]++
		summary.ByTable[res.Conflict.Table]++

		if res.Resolved {
			summary.ResolvedCount++
		} else {
			summary.UnresolvedCount++
		}
	}

	return summary
}
