package main

import (
	"testing"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

func TestGetStrategyForTable(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		config   *config.Config
		expected resolve.Strategy
	}{
		{
			name:     "nil config returns manual",
			table:    "users",
			config:   nil,
			expected: resolve.StrategyManual,
		},
		{
			name:  "default strategy from config",
			table: "users",
			config: &config.Config{
				ConflictResolution: config.ConflictResolutionConfig{
					DefaultStrategy: config.StrategyTheirs,
				},
			},
			expected: resolve.StrategyTheirs,
		},
		{
			name:  "table-specific override",
			table: "logs",
			config: &config.Config{
				ConflictResolution: config.ConflictResolutionConfig{
					DefaultStrategy: config.StrategyManual,
					Strategies: []config.TableStrategy{
						{Table: "logs", Strategy: config.StrategyTheirs},
					},
				},
			},
			expected: resolve.StrategyTheirs,
		},
		{
			name:  "table not in overrides uses default",
			table: "users",
			config: &config.Config{
				ConflictResolution: config.ConflictResolutionConfig{
					DefaultStrategy: config.StrategyOurs,
					Strategies: []config.TableStrategy{
						{Table: "logs", Strategy: config.StrategyTheirs},
					},
				},
			},
			expected: resolve.StrategyOurs,
		},
		{
			name:  "empty default strategy returns manual",
			table: "users",
			config: &config.Config{
				ConflictResolution: config.ConflictResolutionConfig{
					DefaultStrategy: "",
				},
			},
			expected: resolve.StrategyManual,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolve.GetStrategyForTable(tt.table, tt.config)
			if got != tt.expected {
				t.Errorf("GetStrategyForTable(%q) = %v, want %v", tt.table, got, tt.expected)
			}
		})
	}
}

func TestApplyStrategy(t *testing.T) {
	conflict := content.Conflict{
		Table:    "users",
		Key:      "123",
		ProdHash: "abc",
		DevHash:  "xyz",
	}

	tests := []struct {
		name             string
		strategy         resolve.Strategy
		expectedDecision resolve.Decision
		expectedResolved bool
	}{
		{
			name:             "ours strategy keeps prod",
			strategy:         resolve.StrategyOurs,
			expectedDecision: resolve.DecisionKeepProd,
			expectedResolved: true,
		},
		{
			name:             "theirs strategy uses dev",
			strategy:         resolve.StrategyTheirs,
			expectedDecision: resolve.DecisionUseDev,
			expectedResolved: true,
		},
		{
			name:             "manual strategy is pending",
			strategy:         resolve.StrategyManual,
			expectedDecision: resolve.DecisionPending,
			expectedResolved: false,
		},
		{
			name:             "unknown strategy defaults to pending",
			strategy:         resolve.Strategy("unknown"),
			expectedDecision: resolve.DecisionPending,
			expectedResolved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := resolve.ApplyStrategy(conflict, tt.strategy)

			if res.Conflict != conflict {
				t.Errorf("conflict mismatch")
			}
			if res.Strategy != tt.strategy {
				t.Errorf("strategy = %v, want %v", res.Strategy, tt.strategy)
			}
			if res.Decision != tt.expectedDecision {
				t.Errorf("decision = %v, want %v", res.Decision, tt.expectedDecision)
			}
			if res.Resolved != tt.expectedResolved {
				t.Errorf("resolved = %v, want %v", res.Resolved, tt.expectedResolved)
			}
		})
	}
}

func TestResolveConflicts(t *testing.T) {
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "a", DevHash: "b"},
			{Table: "logs", Key: "2", ProdHash: "c", DevHash: "d"},
			{Table: "config", Key: "3", ProdHash: "e", DevHash: "f"},
		},
	}

	cfg := &config.Config{
		ConflictResolution: config.ConflictResolutionConfig{
			DefaultStrategy: config.StrategyManual,
			Strategies: []config.TableStrategy{
				{Table: "logs", Strategy: config.StrategyTheirs},
				{Table: "config", Strategy: config.StrategyOurs},
			},
		},
	}

	resolutions := resolve.ResolveConflicts(conflicts, cfg)

	if len(resolutions) != 3 {
		t.Fatalf("expected 3 resolutions, got %d", len(resolutions))
	}

	// Check each resolution
	for _, res := range resolutions {
		switch res.Conflict.Table {
		case "users":
			if res.Strategy != resolve.StrategyManual {
				t.Errorf("users should have manual strategy, got %v", res.Strategy)
			}
			if res.Resolved {
				t.Error("users should not be resolved")
			}
		case "logs":
			if res.Strategy != resolve.StrategyTheirs {
				t.Errorf("logs should have theirs strategy, got %v", res.Strategy)
			}
			if !res.Resolved {
				t.Error("logs should be resolved")
			}
			if res.Decision != resolve.DecisionUseDev {
				t.Errorf("logs decision should be use_dev, got %v", res.Decision)
			}
		case "config":
			if res.Strategy != resolve.StrategyOurs {
				t.Errorf("config should have ours strategy, got %v", res.Strategy)
			}
			if !res.Resolved {
				t.Error("config should be resolved")
			}
			if res.Decision != resolve.DecisionKeepProd {
				t.Errorf("config decision should be keep_prod, got %v", res.Decision)
			}
		}
	}
}

func TestResolveConflictsWithNilConfig(t *testing.T) {
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "a", DevHash: "b"},
		},
	}

	resolutions := resolve.ResolveConflicts(conflicts, nil)

	if len(resolutions) != 1 {
		t.Fatalf("expected 1 resolution, got %d", len(resolutions))
	}

	if resolutions[0].Strategy != resolve.StrategyManual {
		t.Errorf("expected manual strategy with nil config, got %v", resolutions[0].Strategy)
	}
	if resolutions[0].Resolved {
		t.Error("expected unresolved with nil config")
	}
}

func TestFilterResolved(t *testing.T) {
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "a", DevHash: "b"},
			{Table: "logs", Key: "2", ProdHash: "c", DevHash: "d"},
			{Table: "config", Key: "3", ProdHash: "e", DevHash: "f"},
		},
	}

	resolutions := []resolve.Resolution{
		{Conflict: conflicts.Conflicts[0], Strategy: resolve.StrategyManual, Resolved: false},
		{Conflict: conflicts.Conflicts[1], Strategy: resolve.StrategyTheirs, Resolved: true},
		{Conflict: conflicts.Conflicts[2], Strategy: resolve.StrategyOurs, Resolved: true},
	}

	unresolved := resolve.FilterResolved(conflicts, resolutions)

	if len(unresolved.Conflicts) != 1 {
		t.Fatalf("expected 1 unresolved conflict, got %d", len(unresolved.Conflicts))
	}

	if unresolved.Conflicts[0].Table != "users" {
		t.Errorf("expected users table, got %s", unresolved.Conflicts[0].Table)
	}
}

func TestFilterUnresolved(t *testing.T) {
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1", ProdHash: "a", DevHash: "b"},
			{Table: "logs", Key: "2", ProdHash: "c", DevHash: "d"},
			{Table: "config", Key: "3", ProdHash: "e", DevHash: "f"},
		},
	}

	resolutions := []resolve.Resolution{
		{Conflict: conflicts.Conflicts[0], Strategy: resolve.StrategyManual, Resolved: false},
		{Conflict: conflicts.Conflicts[1], Strategy: resolve.StrategyTheirs, Resolved: true},
		{Conflict: conflicts.Conflicts[2], Strategy: resolve.StrategyOurs, Resolved: true},
	}

	resolved := resolve.FilterUnresolved(conflicts, resolutions)

	if len(resolved.Conflicts) != 2 {
		t.Fatalf("expected 2 resolved conflicts, got %d", len(resolved.Conflicts))
	}
}

func TestCountByDecision(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Decision: resolve.DecisionKeepProd},
		{Decision: resolve.DecisionKeepProd},
		{Decision: resolve.DecisionUseDev},
		{Decision: resolve.DecisionPending},
		{Decision: resolve.DecisionPending},
		{Decision: resolve.DecisionPending},
	}

	counts := resolve.CountByDecision(resolutions)

	if counts[resolve.DecisionKeepProd] != 2 {
		t.Errorf("expected 2 keep_prod, got %d", counts[resolve.DecisionKeepProd])
	}
	if counts[resolve.DecisionUseDev] != 1 {
		t.Errorf("expected 1 use_dev, got %d", counts[resolve.DecisionUseDev])
	}
	if counts[resolve.DecisionPending] != 3 {
		t.Errorf("expected 3 pending, got %d", counts[resolve.DecisionPending])
	}
}

func TestCountByStrategy(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Strategy: resolve.StrategyOurs},
		{Strategy: resolve.StrategyTheirs},
		{Strategy: resolve.StrategyTheirs},
		{Strategy: resolve.StrategyManual},
	}

	counts := resolve.CountByStrategy(resolutions)

	if counts[resolve.StrategyOurs] != 1 {
		t.Errorf("expected 1 ours, got %d", counts[resolve.StrategyOurs])
	}
	if counts[resolve.StrategyTheirs] != 2 {
		t.Errorf("expected 2 theirs, got %d", counts[resolve.StrategyTheirs])
	}
	if counts[resolve.StrategyManual] != 1 {
		t.Errorf("expected 1 manual, got %d", counts[resolve.StrategyManual])
	}
}

func TestGetResolutionsForTable(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "users", Key: "1"}},
		{Conflict: content.Conflict{Table: "users", Key: "2"}},
		{Conflict: content.Conflict{Table: "logs", Key: "3"}},
	}

	userResolutions := resolve.GetResolutionsForTable(resolutions, "users")

	if len(userResolutions) != 2 {
		t.Fatalf("expected 2 user resolutions, got %d", len(userResolutions))
	}

	for _, res := range userResolutions {
		if res.Conflict.Table != "users" {
			t.Errorf("expected users table, got %s", res.Conflict.Table)
		}
	}
}

func TestGetResolvedResolutions(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Resolved: true},
		{Resolved: false},
		{Resolved: true},
	}

	resolved := resolve.GetResolvedResolutions(resolutions)

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved, got %d", len(resolved))
	}

	for _, res := range resolved {
		if !res.Resolved {
			t.Error("expected all to be resolved")
		}
	}
}

func TestGetUnresolvedResolutions(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Resolved: true},
		{Resolved: false},
		{Resolved: true},
	}

	unresolved := resolve.GetUnresolvedResolutions(resolutions)

	if len(unresolved) != 1 {
		t.Fatalf("expected 1 unresolved, got %d", len(unresolved))
	}

	if unresolved[0].Resolved {
		t.Error("expected to be unresolved")
	}
}

func TestEmptyConflicts(t *testing.T) {
	conflicts := content.Conflicts{}

	cfg := &config.Config{
		ConflictResolution: config.ConflictResolutionConfig{
			DefaultStrategy: config.StrategyTheirs,
		},
	}

	resolutions := resolve.ResolveConflicts(conflicts, cfg)

	if len(resolutions) != 0 {
		t.Errorf("expected 0 resolutions for empty conflicts, got %d", len(resolutions))
	}
}

func TestFilterResolvedWithEmptyResolutions(t *testing.T) {
	conflicts := content.Conflicts{
		Conflicts: []content.Conflict{
			{Table: "users", Key: "1"},
			{Table: "logs", Key: "2"},
		},
	}

	unresolved := resolve.FilterResolved(conflicts, []resolve.Resolution{})

	if len(unresolved.Conflicts) != 2 {
		t.Errorf("expected all conflicts unresolved, got %d", len(unresolved.Conflicts))
	}
}

func TestFilterDataDiffByResolutions(t *testing.T) {
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "products",
				Added:   []string{"5", "6"},
				Removed: []string{"7"},
				Updated: []string{"1", "2", "3"},
			},
			{
				Table:   "orders",
				Added:   []string{"10"},
				Removed: []string{},
				Updated: []string{"4", "5"},
			},
			{
				Table:   "customers",
				Updated: []string{"1", "2"},
			},
		},
	}

	resolutions := []resolve.Resolution{
		// products: 1 = theirs (include), 2 = ours (exclude), 3 = manual (exclude)
		{Conflict: content.Conflict{Table: "products", Key: "1"}, Decision: resolve.DecisionUseDev},
		{Conflict: content.Conflict{Table: "products", Key: "2"}, Decision: resolve.DecisionKeepProd},
		{Conflict: content.Conflict{Table: "products", Key: "3"}, Decision: resolve.DecisionPending},
		// orders: 4 = theirs (include), 5 = ours (exclude)
		{Conflict: content.Conflict{Table: "orders", Key: "4"}, Decision: resolve.DecisionUseDev},
		{Conflict: content.Conflict{Table: "orders", Key: "5"}, Decision: resolve.DecisionKeepProd},
		// customers: 1 = manual (exclude), 2 = manual (exclude)
		{Conflict: content.Conflict{Table: "customers", Key: "1"}, Decision: resolve.DecisionPending},
		{Conflict: content.Conflict{Table: "customers", Key: "2"}, Decision: resolve.DecisionPending},
	}

	filtered, excludedCounts := resolve.FilterDataDiffByResolutions(diff, resolutions)

	// Check products table
	var productsDiff *content.TableDataDiff
	for i := range filtered.Tables {
		if filtered.Tables[i].Table == "products" {
			productsDiff = &filtered.Tables[i]
			break
		}
	}
	if productsDiff == nil {
		t.Fatal("products table not found in filtered diff")
	}
	if len(productsDiff.Added) != 2 {
		t.Errorf("products.Added should be unchanged, got %d", len(productsDiff.Added))
	}
	if len(productsDiff.Removed) != 1 {
		t.Errorf("products.Removed should be unchanged, got %d", len(productsDiff.Removed))
	}
	if len(productsDiff.Updated) != 1 || productsDiff.Updated[0] != "1" {
		t.Errorf("products.Updated should only contain '1', got %v", productsDiff.Updated)
	}

	// Check orders table
	var ordersDiff *content.TableDataDiff
	for i := range filtered.Tables {
		if filtered.Tables[i].Table == "orders" {
			ordersDiff = &filtered.Tables[i]
			break
		}
	}
	if ordersDiff == nil {
		t.Fatal("orders table not found in filtered diff")
	}
	if len(ordersDiff.Updated) != 1 || ordersDiff.Updated[0] != "4" {
		t.Errorf("orders.Updated should only contain '4', got %v", ordersDiff.Updated)
	}

	// Check customers table (all excluded)
	var customersDiff *content.TableDataDiff
	for i := range filtered.Tables {
		if filtered.Tables[i].Table == "customers" {
			customersDiff = &filtered.Tables[i]
			break
		}
	}
	if customersDiff == nil {
		t.Fatal("customers table not found in filtered diff")
	}
	if len(customersDiff.Updated) != 0 {
		t.Errorf("customers.Updated should be empty, got %v", customersDiff.Updated)
	}

	// Check excluded counts
	if excludedCounts[resolve.DecisionKeepProd] != 2 {
		t.Errorf("expected 2 keep_prod excluded, got %d", excludedCounts[resolve.DecisionKeepProd])
	}
	if excludedCounts[resolve.DecisionPending] != 3 {
		t.Errorf("expected 3 pending excluded, got %d", excludedCounts[resolve.DecisionPending])
	}
}

func TestFilterDataDiffByResolutionsNoResolutions(t *testing.T) {
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "products",
				Updated: []string{"1", "2", "3"},
			},
		},
	}

	// No resolutions - all keys should remain (backward compatible)
	filtered, excludedCounts := resolve.FilterDataDiffByResolutions(diff, []resolve.Resolution{})

	if len(filtered.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(filtered.Tables))
	}
	if len(filtered.Tables[0].Updated) != 3 {
		t.Errorf("all keys should remain without resolutions, got %d", len(filtered.Tables[0].Updated))
	}
	if len(excludedCounts) != 0 {
		t.Errorf("no keys should be excluded without resolutions")
	}
}

func TestFilterDataDiffByResolutionsPartialResolutions(t *testing.T) {
	diff := content.DataDiff{
		Tables: []content.TableDataDiff{
			{
				Table:   "products",
				Updated: []string{"1", "2", "3", "4"},
			},
		},
	}

	// Only some keys have resolutions
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "products", Key: "1"}, Decision: resolve.DecisionUseDev},
		{Conflict: content.Conflict{Table: "products", Key: "2"}, Decision: resolve.DecisionKeepProd},
		// Keys 3 and 4 have no resolution - should be included
	}

	filtered, excludedCounts := resolve.FilterDataDiffByResolutions(diff, resolutions)

	// Keys 1, 3, 4 should be included; key 2 excluded
	if len(filtered.Tables[0].Updated) != 3 {
		t.Errorf("expected 3 keys (1, 3, 4), got %d: %v", len(filtered.Tables[0].Updated), filtered.Tables[0].Updated)
	}
	if excludedCounts[resolve.DecisionKeepProd] != 1 {
		t.Errorf("expected 1 keep_prod excluded, got %d", excludedCounts[resolve.DecisionKeepProd])
	}
}

func TestBuildResolutionSummary(t *testing.T) {
	resolutions := []resolve.Resolution{
		{Conflict: content.Conflict{Table: "products", Key: "1"}, Strategy: resolve.StrategyTheirs, Decision: resolve.DecisionUseDev, Resolved: true},
		{Conflict: content.Conflict{Table: "products", Key: "2"}, Strategy: resolve.StrategyOurs, Decision: resolve.DecisionKeepProd, Resolved: true},
		{Conflict: content.Conflict{Table: "orders", Key: "1"}, Strategy: resolve.StrategyManual, Decision: resolve.DecisionPending, Resolved: false},
		{Conflict: content.Conflict{Table: "customers", Key: "1"}, Strategy: resolve.StrategyManual, Decision: resolve.DecisionPending, Resolved: false},
	}

	summary := resolve.BuildResolutionSummary(resolutions)

	if summary.TotalConflicts != 4 {
		t.Errorf("expected 4 total conflicts, got %d", summary.TotalConflicts)
	}
	if summary.ResolvedCount != 2 {
		t.Errorf("expected 2 resolved, got %d", summary.ResolvedCount)
	}
	if summary.UnresolvedCount != 2 {
		t.Errorf("expected 2 unresolved, got %d", summary.UnresolvedCount)
	}
	if summary.ByStrategy[resolve.StrategyTheirs] != 1 {
		t.Errorf("expected 1 theirs, got %d", summary.ByStrategy[resolve.StrategyTheirs])
	}
	if summary.ByStrategy[resolve.StrategyOurs] != 1 {
		t.Errorf("expected 1 ours, got %d", summary.ByStrategy[resolve.StrategyOurs])
	}
	if summary.ByStrategy[resolve.StrategyManual] != 2 {
		t.Errorf("expected 2 manual, got %d", summary.ByStrategy[resolve.StrategyManual])
	}
	if summary.ByTable["products"] != 2 {
		t.Errorf("expected 2 products conflicts, got %d", summary.ByTable["products"])
	}
	if summary.ByTable["orders"] != 1 {
		t.Errorf("expected 1 orders conflict, got %d", summary.ByTable["orders"])
	}
}

func TestBuildResolutionSummaryEmpty(t *testing.T) {
	summary := resolve.BuildResolutionSummary([]resolve.Resolution{})

	if summary.TotalConflicts != 0 {
		t.Errorf("expected 0 total conflicts, got %d", summary.TotalConflicts)
	}
	if summary.ResolvedCount != 0 {
		t.Errorf("expected 0 resolved, got %d", summary.ResolvedCount)
	}
}
