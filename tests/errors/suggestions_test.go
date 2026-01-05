package errors_test

import (
	"strings"
	"testing"

	"github.com/iamvirul/deepdiff-db/pkg/errors"
)

func TestGetSuggestions(t *testing.T) {
	tests := []struct {
		name         string
		code         errors.ErrorCode
		context      map[string]any
		wantContains []string
	}{
		{
			name:    "connection_failed",
			code:    errors.ErrConnectionFailed,
			context: nil,
			wantContains: []string{
				"Verify database host and port",
				"Check network connectivity",
			},
		},
		{
			name:    "missing_primary_key",
			code:    errors.ErrMissingPrimaryKey,
			context: nil,
			wantContains: []string{
				"Add a primary key to the table",
				"Use --ignore-tables",
			},
		},
		{
			name: "connection_failed_with_host",
			code: errors.ErrConnectionFailed,
			context: map[string]any{
				"host": "localhost",
				"port": 3306,
			},
			wantContains: []string{
				"Verify you can reach localhost:3306",
			},
		},
		{
			name: "missing_primary_key_with_table",
			code: errors.ErrMissingPrimaryKey,
			context: map[string]any{
				"table": "users",
			},
			wantContains: []string{
				"Add primary key to table 'users'",
				"Or add 'users' to ignore_tables",
			},
		},
		{
			name: "pack_application_with_statement",
			code: errors.ErrPackApplication,
			context: map[string]any{
				"statement":       "INSERT INTO users VALUES (1, 'test')",
				"statement_index": 5,
			},
			wantContains: []string{
				"Review failed SQL:",
				"Error occurred at statement #5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := errors.GetSuggestions(tt.code, tt.context)

			if len(suggestions) == 0 {
				t.Error("expected at least one suggestion")
			}

			for _, want := range tt.wantContains {
				found := false
				for _, suggestion := range suggestions {
					if strings.Contains(suggestion, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected suggestions to contain %q, got: %v", want, suggestions)
				}
			}
		})
	}
}

func TestAddDefaultSuggestions(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed")

	// Before adding suggestions
	if len(err.Suggestions) != 0 {
		t.Error("expected no suggestions initially")
	}

	// Add default suggestions
	err = errors.AddDefaultSuggestions(err)

	// Should now have suggestions
	if len(err.Suggestions) == 0 {
		t.Error("expected suggestions after AddDefaultSuggestions")
	}

	// Verify they're the right ones
	found := false
	for _, s := range err.Suggestions {
		if strings.Contains(s, "Verify database host and port") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected default suggestion for connection failed, got: %v", err.Suggestions)
	}
}

func TestAddDefaultSuggestions_NoDuplicates(t *testing.T) {
	err := errors.New(errors.ErrConnectionFailed, "connection failed").
		WithSuggestion("Check network connectivity")

	// Add default suggestions (which includes "Check network connectivity")
	err = errors.AddDefaultSuggestions(err)

	// Count occurrences
	count := 0
	for _, s := range err.Suggestions {
		if s == "Check network connectivity" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("expected suggestion to appear once, appeared %d times", count)
	}
}

func TestAddDefaultSuggestionsNil(t *testing.T) {
	var err *errors.Error
	result := errors.AddDefaultSuggestions(err)
	if result != nil {
		t.Error("AddDefaultSuggestions on nil error should return nil")
	}
}

func TestSuggestionsAllErrorCodes(t *testing.T) {
	// Verify that we have suggestions for common error codes
	errorCodes := []errors.ErrorCode{
		errors.ErrConnectionFailed,
		errors.ErrMissingPrimaryKey,
		errors.ErrSchemaDrift,
		errors.ErrHashingFailed,
		errors.ErrPackApplication,
		errors.ErrCheckpointInvalid,
		errors.ErrConfigInvalid,
		errors.ErrOutOfMemory,
		errors.ErrDiskFull,
	}

	for _, code := range errorCodes {
		t.Run(string(code), func(t *testing.T) {
			suggestions := errors.GetSuggestions(code, nil)
			if len(suggestions) == 0 {
				t.Errorf("expected suggestions for error code %s", code)
			}
		})
	}
}

func TestContextualSuggestions(t *testing.T) {
	tests := []struct {
		name    string
		code    errors.ErrorCode
		context map[string]any
		checkFunc func([]string) bool
	}{
		{
			name: "checkpoint_invalid_with_path",
			code: errors.ErrCheckpointInvalid,
			context: map[string]any{
				"path": "/tmp/checkpoint.json",
			},
			checkFunc: func(suggestions []string) bool {
				for _, s := range suggestions {
					if strings.Contains(s, "/tmp/checkpoint.json") {
						return true
					}
				}
				return false
			},
		},
		{
			name: "pack_application_with_long_statement",
			code: errors.ErrPackApplication,
			context: map[string]any{
				"statement": strings.Repeat("A", 200),
			},
			checkFunc: func(suggestions []string) bool {
				for _, s := range suggestions {
					// Should truncate long statements
					if strings.Contains(s, "...") && len(s) < 150 {
						return true
					}
				}
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := errors.GetSuggestions(tt.code, tt.context)
			if !tt.checkFunc(suggestions) {
				t.Errorf("contextual suggestion check failed for %s, got: %v", tt.name, suggestions)
			}
		})
	}
}
