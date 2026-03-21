package content

import (
	"path/filepath"
	"strings"
)

// IgnoreMatcher returns a function to check if a column should be ignored.
// IgnoreMatcher returns a predicate that reports whether a given table and column pair should be ignored
// according to the provided columnPatterns.
//
// The returned predicate is case-insensitive and treats each entry in columnPatterns as either an exact
// pattern ("table.column" or "column") or a glob pattern (shell-style wildcards). A match occurs if a pattern
// equals the full "table.column", equals just "column", or glob-matches either the full name or the column
// name. Invalid glob patterns are ignored (treated as non-matching).
//
// columnPatterns is the list of patterns to test against.
func IgnoreMatcher(columnPatterns []string) func(table, column string) bool {
	normalized := make([]string, 0, len(columnPatterns))
	for _, p := range columnPatterns {
		normalized = append(normalized, strings.ToLower(p))
	}
	return func(table, column string) bool {
		full := strings.ToLower(table + "." + column)
		col := strings.ToLower(column)
		for _, pat := range normalized {
			// exact
			if pat == full || pat == col {
				return true
			}
			// glob
			if ok, _ := filepath.Match(pat, full); ok {
				return true
			}
			if ok, _ := filepath.Match(pat, col); ok {
				return true
			}
		}
		return false
	}
}
