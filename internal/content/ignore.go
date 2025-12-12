package content

import (
	"path/filepath"
	"strings"
)

// IgnoreMatcher returns a function to check if a column should be ignored.
// Column patterns may include glob wildcards and can be in form "table.column" or "*.col".
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
