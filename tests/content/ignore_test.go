package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"testing"
)

func TestIgnoreMatcher(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		table    string
		column   string
		expected bool
	}{
		{
			name:     "exact match table.column",
			patterns: []string{"users.email"},
			table:    "users",
			column:   "email",
			expected: true,
		},
		{
			name:     "exact match column only",
			patterns: []string{"updated_at"},
			table:    "users",
			column:   "updated_at",
			expected: true,
		},
		{
			name:     "glob match *.column",
			patterns: []string{"*.updated_at"},
			table:    "users",
			column:   "updated_at",
			expected: true,
		},
		{
			name:     "glob match *.column different table",
			patterns: []string{"*.updated_at"},
			table:    "posts",
			column:   "updated_at",
			expected: true,
		},
		{
			name:     "no match",
			patterns: []string{"*.updated_at"},
			table:    "users",
			column:   "email",
			expected: false,
		},
		{
			name:     "case insensitive",
			patterns: []string{"USERS.EMAIL"},
			table:    "users",
			column:   "email",
			expected: true,
		},
		{
			name:     "case insensitive column",
			patterns: []string{"UPDATED_AT"},
			table:    "users",
			column:   "updated_at",
			expected: true,
		},
		{
			name:     "multiple patterns match",
			patterns: []string{"*.updated_at", "*.created_at"},
			table:    "users",
			column:   "updated_at",
			expected: true,
		},
		{
			name:     "multiple patterns no match",
			patterns: []string{"*.updated_at", "*.created_at"},
			table:    "users",
			column:   "email",
			expected: false,
		},
		{
			name:     "empty patterns",
			patterns: []string{},
			table:    "users",
			column:   "email",
			expected: false,
		},
		{
			name:     "glob pattern with wildcard",
			patterns: []string{"users.*"},
			table:    "users",
			column:   "email",
			expected: true,
		},
		{
			name:     "glob pattern with wildcard different table",
			patterns: []string{"users.*"},
			table:    "posts",
			column:   "email",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := content.IgnoreMatcher(tt.patterns)
			result := matcher(tt.table, tt.column)
			if result != tt.expected {
				t.Errorf("content.IgnoreMatcher(%v)(%q, %q) = %v, want %v",
					tt.patterns, tt.table, tt.column, result, tt.expected)
			}
		})
	}
}

func TestIgnoreMatcher_RealWorldScenarios(t *testing.T) {
	patterns := []string{
		"*.updated_at",
		"*.created_at",
		"logs.*",
		"audit.*",
	}

	matcher := content.IgnoreMatcher(patterns)

	// Should ignore timestamp columns in any table
	if !matcher("users", "updated_at") {
		t.Error("expected users.updated_at to be ignored")
	}
	if !matcher("posts", "created_at") {
		t.Error("expected posts.created_at to be ignored")
	}

	// Should ignore all columns in logs table
	if !matcher("logs", "message") {
		t.Error("expected logs.message to be ignored")
	}
	if !matcher("logs", "level") {
		t.Error("expected logs.level to be ignored")
	}

	// Should not ignore regular columns
	if matcher("users", "email") {
		t.Error("expected users.email to not be ignored")
	}
	if matcher("posts", "title") {
		t.Error("expected posts.title to not be ignored")
	}
}
