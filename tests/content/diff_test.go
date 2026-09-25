package main

import (
	"github.com/iamvirul/deepdiff-db/internal/content"

	"testing"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

func TestDiffTableHashes(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		prod     map[string]string
		dev      map[string]string
		validate func(*testing.T, content.TableDataDiff)
	}{
		{
			name:  "identical hashes",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
				"2": "hash2",
			},
			dev: map[string]string{
				"1": "hash1",
				"2": "hash2",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if td.Table != "users" {
					t.Errorf("expected table 'users', got %s", td.Table)
				}
				if len(td.Added) != 0 {
					t.Errorf("expected no added rows, got %v", td.Added)
				}
				if len(td.Removed) != 0 {
					t.Errorf("expected no removed rows, got %v", td.Removed)
				}
				if len(td.Updated) != 0 {
					t.Errorf("expected no updated rows, got %v", td.Updated)
				}
			},
		},
		{
			name:  "added rows",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
			},
			dev: map[string]string{
				"1": "hash1",
				"2": "hash2",
				"3": "hash3",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Added) != 2 {
					t.Errorf("expected 2 added rows, got %d", len(td.Added))
				}
				if len(td.Removed) != 0 {
					t.Errorf("expected no removed rows, got %v", td.Removed)
				}
				if len(td.Updated) != 0 {
					t.Errorf("expected no updated rows, got %v", td.Updated)
				}
			},
		},
		{
			name:  "removed rows",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
				"2": "hash2",
				"3": "hash3",
			},
			dev: map[string]string{
				"1": "hash1",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Added) != 0 {
					t.Errorf("expected no added rows, got %v", td.Added)
				}
				if len(td.Removed) != 2 {
					t.Errorf("expected 2 removed rows, got %d", len(td.Removed))
				}
				if len(td.Updated) != 0 {
					t.Errorf("expected no updated rows, got %v", td.Updated)
				}
			},
		},
		{
			name:  "updated rows",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
				"2": "hash2_old",
			},
			dev: map[string]string{
				"1": "hash1",
				"2": "hash2_new",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Added) != 0 {
					t.Errorf("expected no added rows, got %v", td.Added)
				}
				if len(td.Removed) != 0 {
					t.Errorf("expected no removed rows, got %v", td.Removed)
				}
				if len(td.Updated) != 1 {
					t.Errorf("expected 1 updated row, got %d", len(td.Updated))
				}
				if td.Updated[0] != "2" {
					t.Errorf("expected updated key '2', got %s", td.Updated[0])
				}
			},
		},
		{
			name:  "mixed changes",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
				"2": "hash2_old",
				"3": "hash3",
			},
			dev: map[string]string{
				"1": "hash1",
				"2": "hash2_new",
				"4": "hash4",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Added) != 1 {
					t.Errorf("expected 1 added row, got %d", len(td.Added))
				}
				if len(td.Removed) != 1 {
					t.Errorf("expected 1 removed row, got %d", len(td.Removed))
				}
				if len(td.Updated) != 1 {
					t.Errorf("expected 1 updated row, got %d", len(td.Updated))
				}
			},
		},
		{
			name:  "empty prod",
			table: "users",
			prod:  map[string]string{},
			dev: map[string]string{
				"1": "hash1",
				"2": "hash2",
			},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Added) != 2 {
					t.Errorf("expected 2 added rows, got %d", len(td.Added))
				}
			},
		},
		{
			name:  "empty dev",
			table: "users",
			prod: map[string]string{
				"1": "hash1",
				"2": "hash2",
			},
			dev: map[string]string{},
			validate: func(t *testing.T, td content.TableDataDiff) {
				if len(td.Removed) != 2 {
					t.Errorf("expected 2 removed rows, got %d", len(td.Removed))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := content.DiffTableHashes(tt.table, tt.prod, tt.dev)
			tt.validate(t, result)
		})
	}
}

func TestBuildDataDiff(t *testing.T) {
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users"},
			"posts": {Name: "posts"},
		},
	}

	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"users": {Name: "users"},
			"posts": {Name: "posts"},
		},
	}

	prodHashes := map[string]map[string]string{
		"users": {
			"1": "hash1",
			"2": "hash2_old",
		},
		"posts": {
			"1": "hash1",
		},
	}

	devHashes := map[string]map[string]string{
		"users": {
			"1": "hash1",
			"2": "hash2_new",
			"3": "hash3",
		},
		"posts": {
			"1": "hash1",
		},
	}

	diff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	if len(diff.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(diff.Tables))
	}

	// Check users table
	var usersDiff *content.TableDataDiff
	for i := range diff.Tables {
		if diff.Tables[i].Table == "users" {
			usersDiff = &diff.Tables[i]
			break
		}
	}
	if usersDiff == nil {
		t.Fatal("users table diff not found")
	}
	if len(usersDiff.Added) != 1 || usersDiff.Added[0] != "3" {
		t.Errorf("expected added key '3', got %v", usersDiff.Added)
	}
	if len(usersDiff.Updated) != 1 || usersDiff.Updated[0] != "2" {
		t.Errorf("expected updated key '2', got %v", usersDiff.Updated)
	}

	// Check conflicts
	if len(conflicts.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts.Conflicts))
	}
	conflict := conflicts.Conflicts[0]
	if conflict.Table != "users" || conflict.Key != "2" {
		t.Errorf("expected conflict on users.2, got %s.%s", conflict.Table, conflict.Key)
	}
	if !conflicts.HasConflicts() {
		t.Error("expected HasConflicts() to return true")
	}
}

// TestBuildDataDiff_CaseInsensitiveTableMatch verifies that a table which a
// cross-engine migration folds to a different case (PostgreSQL "customers" ->
// Oracle "CUSTOMERS") is still matched, so missing and changed rows are
// detected instead of the table being silently skipped.
func TestBuildDataDiff_CaseInsensitiveTableMatch(t *testing.T) {
	prodSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"customers": {Name: "customers"},
		},
	}
	devSchema := &schema.Schema{
		Tables: map[string]schema.Table{
			"CUSTOMERS": {Name: "CUSTOMERS"},
		},
	}

	// Prod (source of truth) has ids 1,2,5; dev (Oracle) has 1,2 with 2 changed.
	prodHashes := map[string]map[string]string{
		"customers": {"1": "h1", "2": "h2_old", "5": "h5"},
	}
	devHashes := map[string]map[string]string{
		"CUSTOMERS": {"1": "h1", "2": "h2_new"},
	}

	diff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	if len(diff.Tables) != 1 {
		t.Fatalf("expected 1 matched table, got %d", len(diff.Tables))
	}
	td := diff.Tables[0]
	if td.Table != "customers" {
		t.Errorf("expected prod-side label 'customers', got %q", td.Table)
	}
	if len(td.Removed) != 1 || td.Removed[0] != "5" {
		t.Errorf("expected removed (missing in dev) [5], got %v", td.Removed)
	}
	if len(td.Updated) != 1 || td.Updated[0] != "2" {
		t.Errorf("expected updated [2], got %v", td.Updated)
	}
	if len(td.Added) != 0 {
		t.Errorf("expected no added rows, got %v", td.Added)
	}
	if len(conflicts.Conflicts) != 1 || conflicts.Conflicts[0].Table != "customers" || conflicts.Conflicts[0].Key != "2" {
		t.Errorf("expected 1 conflict on customers.2, got %v", conflicts.Conflicts)
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name     string
		diff     content.DataDiff
		expected bool
	}{
		{
			name: "no changes",
			diff: content.DataDiff{
				Tables: []content.TableDataDiff{
					{Table: "users"},
				},
			},
			expected: false,
		},
		{
			name: "has added",
			diff: content.DataDiff{
				Tables: []content.TableDataDiff{
					{Table: "users", Added: []string{"1"}},
				},
			},
			expected: true,
		},
		{
			name: "has removed",
			diff: content.DataDiff{
				Tables: []content.TableDataDiff{
					{Table: "users", Removed: []string{"1"}},
				},
			},
			expected: true,
		},
		{
			name: "has updated",
			diff: content.DataDiff{
				Tables: []content.TableDataDiff{
					{Table: "users", Updated: []string{"1"}},
				},
			},
			expected: true,
		},
		{
			name: "multiple changes",
			diff: content.DataDiff{
				Tables: []content.TableDataDiff{
					{Table: "users", Added: []string{"1"}, Removed: []string{"2"}, Updated: []string{"3"}},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.HasChanges(); got != tt.expected {
				t.Errorf("HasChanges() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHasConflicts(t *testing.T) {
	tests := []struct {
		name      string
		conflicts content.Conflicts
		expected  bool
	}{
		{
			name:      "no conflicts",
			conflicts: content.Conflicts{Conflicts: []content.Conflict{}},
			expected:  false,
		},
		{
			name: "has conflicts",
			conflicts: content.Conflicts{
				Conflicts: []content.Conflict{
					{Table: "users", Key: "1"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conflicts.HasConflicts(); got != tt.expected {
				t.Errorf("HasConflicts() = %v, want %v", got, tt.expected)
			}
		})
	}
}
