package errors_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

func TestGetSuggestions_Default(t *testing.T) {
	suggestions := errors.GetSuggestions(errors.ErrConnectionFailed, nil)

	if len(suggestions) == 0 {
		t.Fatal("expected default suggestions")
	}

	// Check some expected suggestions
	hasNetwork := false
	for _, s := range suggestions {
		if strings.Contains(strings.ToLower(s), "network") {
			hasNetwork = true
			break
		}
	}
	if !hasNetwork {
		t.Error("expected network-related suggestion")
	}
}

func TestGetSuggestions_WithContext(t *testing.T) {
	context := map[string]any{
		"host": "localhost",
		"port": 3306,
	}

	suggestions := errors.GetSuggestions(errors.ErrConnectionFailed, context)

	// Should have default suggestions plus context-specific ones
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions")
	}

	// Check for context-specific suggestion
	hasContextSuggestion := false
	for _, s := range suggestions {
		if strings.Contains(s, "localhost") && strings.Contains(s, "3306") {
			hasContextSuggestion = true
			break
		}
	}
	if !hasContextSuggestion {
		t.Error("expected context-specific suggestion")
	}
}

func TestGetSuggestions_MissingPrimaryKey(t *testing.T) {
	context := map[string]any{
		"table": "users",
	}

	suggestions := errors.GetSuggestions(errors.ErrMissingPrimaryKey, context)

	hasTableSuggestion := false
	for _, s := range suggestions {
		if strings.Contains(s, "users") {
			hasTableSuggestion = true
			break
		}
	}
	if !hasTableSuggestion {
		t.Error("expected table-specific suggestion")
	}
}

func TestGetSuggestions_PackApplication(t *testing.T) {
	context := map[string]any{
		"statement":       "INSERT INTO users VALUES (1, 'test')",
		"statement_index":  5,
	}

	suggestions := errors.GetSuggestions(errors.ErrPackApplication, context)

	hasStatementSuggestion := false
	hasIndexSuggestion := false
	for _, s := range suggestions {
		if strings.Contains(s, "INSERT") {
			hasStatementSuggestion = true
		}
		if strings.Contains(s, "#5") {
			hasIndexSuggestion = true
		}
	}

	if !hasStatementSuggestion {
		t.Error("expected statement-specific suggestion")
	}
	if !hasIndexSuggestion {
		t.Error("expected index-specific suggestion")
	}
}

func TestGetSuggestions_CheckpointInvalid(t *testing.T) {
	context := map[string]any{
		"path": "/path/to/checkpoint.json",
	}

	suggestions := errors.GetSuggestions(errors.ErrCheckpointInvalid, context)

	hasPathSuggestion := false
	for _, s := range suggestions {
		if strings.Contains(s, "checkpoint.json") {
			hasPathSuggestion = true
			break
		}
	}
	if !hasPathSuggestion {
		t.Error("expected path-specific suggestion")
	}
}

func TestGetSuggestions_UnknownCode(t *testing.T) {
	// Test with a code that doesn't have default suggestions
	suggestions := errors.GetSuggestions(errors.ErrorCode("UNKNOWN_CODE"), nil)

	// Should return empty or context-specific only
	_ = suggestions // Just ensure it doesn't panic
}

func TestAddDefaultSuggestions(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	// Initially no suggestions
	if len(err.Suggestions) != 0 {
		t.Error("expected no suggestions initially")
	}

	// Add default suggestions
	enhanced := errors.AddDefaultSuggestions(err)

	if enhanced == nil {
		t.Fatal("expected non-nil error")
	}

	if len(enhanced.Suggestions) == 0 {
		t.Error("expected suggestions to be added")
	}
}

func TestAddDefaultSuggestions_Nil(t *testing.T) {
	var err *errors.Error
	result := errors.AddDefaultSuggestions(err)

	if result != nil {
		t.Error("expected nil for nil error")
	}
}

func TestAddDefaultSuggestions_NoDuplicates(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		WithSuggestion("Verify database host and port are correct")

	// Add default suggestions (one should already exist)
	enhanced := errors.AddDefaultSuggestions(err)

	// Count occurrences of the first suggestion
	count := 0
	firstSuggestion := enhanced.Suggestions[0]
	for _, s := range enhanced.Suggestions {
		if s == firstSuggestion {
			count++
		}
	}

	if count > 1 {
		t.Error("expected no duplicate suggestions")
	}
}

func TestTruncate(t *testing.T) {
	// Test truncation in context-specific suggestions
	context := map[string]any{
		"statement": strings.Repeat("a", 200), // Long statement
	}

	suggestions := errors.GetSuggestions(errors.ErrPackApplication, context)

	hasTruncated := false
	for _, s := range suggestions {
		if strings.Contains(s, "...") {
			hasTruncated = true
			break
		}
	}
	if !hasTruncated {
		t.Error("expected truncated statement in suggestion")
	}
}

func TestGetSuggestions_AllErrorCodes(t *testing.T) {
	// Test that all error codes have suggestions or handle gracefully
	codes := []errors.ErrorCode{
		errors.ErrConnectionFailed,
		errors.ErrConnectionTimeout,
		errors.ErrAuthenticationFailed,
		errors.ErrDatabaseNotFound,
		errors.ErrQueryFailed,
		errors.ErrSchemaDrift,
		errors.ErrMissingPrimaryKey,
		errors.ErrInvalidSchema,
		errors.ErrColumnMismatch,
		errors.ErrSchemaLoadFailed,
		errors.ErrHashingFailed,
		errors.ErrDataCorruption,
		errors.ErrConflictDetected,
		errors.ErrDataComparison,
		errors.ErrPackGeneration,
		errors.ErrPackApplication,
		errors.ErrTransactionFailed,
		errors.ErrRollbackFailed,
		errors.ErrMigrationValidation,
		errors.ErrCheckpointRead,
		errors.ErrCheckpointWrite,
		errors.ErrCheckpointInvalid,
		errors.ErrResumeStateMismatch,
		errors.ErrCheckpointExpired,
		errors.ErrConfigInvalid,
		errors.ErrConfigMissing,
		errors.ErrConfigParse,
		errors.ErrOutOfMemory,
		errors.ErrDiskFull,
		errors.ErrPermissionDenied,
		errors.ErrFileNotFound,
	}

	for _, code := range codes {
		suggestions := errors.GetSuggestions(code, nil)
		_ = suggestions // Just ensure it doesn't panic
	}
}

