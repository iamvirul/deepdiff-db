package schema

import (
	"fmt"
	"sort"
	"strings"
)

// OperationType represents the type of migration operation.
type OperationType int

const (
	// Operation order priority (lower = earlier in migration)
	OpDropForeignKey OperationType = iota // Must drop FKs before dropping tables/columns
	OpDropIndex                           // Drop indexes before dropping columns
	OpDropColumn                          // Drop columns before dropping tables
	OpDropTable                           // Drop tables last in DROP operations
	OpCreateTable                         // Create tables first in CREATE operations
	OpAddColumn                           // Add columns after tables exist
	OpModifyColumn                        // Modify columns after they exist
	OpAddIndex                            // Add indexes after columns exist
	OpAddForeignKey                       // Add FKs last (referenced tables must exist)
	OpModifyPrimaryKey                    // Modify PKs (complex, usually last)
)

// String returns a human-readable name for the operation type.
func (o OperationType) String() string {
	switch o {
	case OpDropForeignKey:
		return "DROP_FOREIGN_KEY"
	case OpDropIndex:
		return "DROP_INDEX"
	case OpDropColumn:
		return "DROP_COLUMN"
	case OpDropTable:
		return "DROP_TABLE"
	case OpCreateTable:
		return "CREATE_TABLE"
	case OpAddColumn:
		return "ADD_COLUMN"
	case OpModifyColumn:
		return "MODIFY_COLUMN"
	case OpAddIndex:
		return "ADD_INDEX"
	case OpAddForeignKey:
		return "ADD_FOREIGN_KEY"
	case OpModifyPrimaryKey:
		return "MODIFY_PRIMARY_KEY"
	default:
		return "UNKNOWN"
	}
}

// MigrationOperation represents a single migration operation with its dependencies.
type MigrationOperation struct {
	Type       OperationType
	Table      string      // Target table name
	Object     string      // Column, index, or FK name (empty for table operations)
	Data       interface{} // Associated data (Column, Index, ForeignKey, etc.)
	DependsOn  []string    // Operation IDs this operation depends on
	ID         string      // Unique identifier for this operation
	Priority   int         // Used for stable sorting within same type
}

// String returns a human-readable description of the operation.
func (m *MigrationOperation) String() string {
	if m.Object != "" {
		return fmt.Sprintf("%s %s.%s", m.Type.String(), m.Table, m.Object)
	}
	return fmt.Sprintf("%s %s", m.Type.String(), m.Table)
}

// DependencyGraph represents a directed graph of migration operations.
type DependencyGraph struct {
	Operations map[string]*MigrationOperation // ID -> Operation
	Edges      map[string][]string            // ID -> list of dependent operation IDs
	InDegree   map[string]int                 // ID -> number of incoming edges
}

// NewDependencyGraph creates a new empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Operations: make(map[string]*MigrationOperation),
		Edges:      make(map[string][]string),
		InDegree:   make(map[string]int),
	}
}

// AddOperation adds an operation to the graph.
func (g *DependencyGraph) AddOperation(op *MigrationOperation) {
	g.Operations[op.ID] = op
	if _, exists := g.InDegree[op.ID]; !exists {
		g.InDegree[op.ID] = 0
	}
}

// AddDependency adds a dependency edge: 'from' must complete before 'to' can start.
func (g *DependencyGraph) AddDependency(from, to string) {
	// Ensure both nodes exist in InDegree
	if _, exists := g.InDegree[from]; !exists {
		g.InDegree[from] = 0
	}
	if _, exists := g.InDegree[to]; !exists {
		g.InDegree[to] = 0
	}

	g.Edges[from] = append(g.Edges[from], to)
	g.InDegree[to]++
}

// CircularDependency represents a detected circular dependency.
type CircularDependency struct {
	Cycle   []string // Operation IDs forming the cycle
	Tables  []string // Table names involved
	Message string   // Human-readable description
}

// OrderingResult contains the ordered operations and any warnings.
type OrderingResult struct {
	Operations           []*MigrationOperation
	CircularDependencies []CircularDependency
	Warnings             []string
}

// BuildDependencyGraph constructs a dependency graph from a DiffResult.
// It analyzes foreign key relationships and determines the correct order
// for all migration operations.
func BuildDependencyGraph(diff DiffResult, devSchema *Schema) *DependencyGraph {
	g := NewDependencyGraph()

	// Track all table names for FK reference checking
	allTables := make(map[string]bool)
	addedTables := make(map[string]bool)
	removedTables := make(map[string]bool)

	// Collect table information
	for _, t := range diff.AddedTables {
		allTables[t.Name] = true
		addedTables[t.Name] = true
	}
	for _, name := range diff.RemovedTables {
		allTables[name] = true
		removedTables[name] = true
	}
	for _, td := range diff.Tables {
		if !td.OnlyInProd && !td.OnlyInDev {
			allTables[td.Name] = true
		}
	}

	priority := 0

	// === DROP OPERATIONS (reverse dependency order) ===

	// 1. DROP FOREIGN KEYS - must happen before dropping tables they reference
	for _, td := range diff.Tables {
		for _, fk := range td.RemovedForeignKeys {
			op := &MigrationOperation{
				Type:     OpDropForeignKey,
				Table:    td.Name,
				Object:   fk.Name,
				Data:     fk,
				ID:       fmt.Sprintf("drop_fk_%s_%s", td.Name, fk.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)
		}
	}

	// 2. DROP INDEXES - must happen before dropping columns they reference
	for _, td := range diff.Tables {
		for _, idx := range td.RemovedIndexes {
			op := &MigrationOperation{
				Type:     OpDropIndex,
				Table:    td.Name,
				Object:   idx.Name,
				Data:     idx,
				ID:       fmt.Sprintf("drop_idx_%s_%s", td.Name, idx.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)

			// Index must be dropped before any of its columns are dropped
			for _, col := range idx.Columns {
				dropColID := fmt.Sprintf("drop_col_%s_%s", td.Name, col)
				if _, exists := g.Operations[dropColID]; exists {
					g.AddDependency(op.ID, dropColID)
				}
			}
		}
	}

	// 3. DROP COLUMNS
	for _, td := range diff.Tables {
		for _, col := range td.RemovedColumns {
			op := &MigrationOperation{
				Type:     OpDropColumn,
				Table:    td.Name,
				Object:   col.Name,
				Data:     col,
				ID:       fmt.Sprintf("drop_col_%s_%s", td.Name, col.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)
		}
	}

	// 4. DROP TABLES - must happen after FKs referencing them are dropped
	for _, tableName := range diff.RemovedTables {
		op := &MigrationOperation{
			Type:     OpDropTable,
			Table:    tableName,
			Data:     tableName,
			ID:       fmt.Sprintf("drop_table_%s", tableName),
			Priority: priority,
		}
		priority++
		g.AddOperation(op)
	}

	// === CREATE OPERATIONS (dependency order) ===

	// 5. CREATE TABLES - tables must exist before FKs can reference them
	for i, table := range diff.AddedTables {
		op := &MigrationOperation{
			Type:     OpCreateTable,
			Table:    table.Name,
			Data:     table,
			ID:       fmt.Sprintf("create_table_%s", table.Name),
			Priority: priority + i,
		}
		g.AddOperation(op)
	}
	priority += len(diff.AddedTables)

	// 6. ADD COLUMNS
	for _, td := range diff.Tables {
		for _, col := range td.AddedColumns {
			op := &MigrationOperation{
				Type:     OpAddColumn,
				Table:    td.Name,
				Object:   col.Name,
				Data:     col,
				ID:       fmt.Sprintf("add_col_%s_%s", td.Name, col.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)
		}
	}

	// 7. MODIFY COLUMNS
	for _, td := range diff.Tables {
		for _, colDiff := range td.ModifiedColumns {
			op := &MigrationOperation{
				Type:     OpModifyColumn,
				Table:    td.Name,
				Object:   colDiff.Column,
				Data:     colDiff,
				ID:       fmt.Sprintf("mod_col_%s_%s", td.Name, colDiff.Column),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)
		}
	}

	// 8. ADD INDEXES - columns must exist first
	for _, td := range diff.Tables {
		for _, idx := range td.AddedIndexes {
			op := &MigrationOperation{
				Type:     OpAddIndex,
				Table:    td.Name,
				Object:   idx.Name,
				Data:     idx,
				ID:       fmt.Sprintf("add_idx_%s_%s", td.Name, idx.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)

			// Index depends on its columns being added first
			for _, col := range idx.Columns {
				addColID := fmt.Sprintf("add_col_%s_%s", td.Name, col)
				if _, exists := g.Operations[addColID]; exists {
					g.AddDependency(addColID, op.ID)
				}
			}
		}
	}

	// Also add indexes for new tables
	for _, table := range diff.AddedTables {
		createTableID := fmt.Sprintf("create_table_%s", table.Name)
		for _, idx := range table.Indexes {
			op := &MigrationOperation{
				Type:     OpAddIndex,
				Table:    table.Name,
				Object:   idx.Name,
				Data:     idx,
				ID:       fmt.Sprintf("add_idx_%s_%s", table.Name, idx.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)

			// Index depends on table being created
			g.AddDependency(createTableID, op.ID)
		}
	}

	// 9. ADD FOREIGN KEYS - referenced tables must exist
	for _, td := range diff.Tables {
		for _, fk := range td.AddedForeignKeys {
			op := &MigrationOperation{
				Type:     OpAddForeignKey,
				Table:    td.Name,
				Object:   fk.Name,
				Data:     fk,
				ID:       fmt.Sprintf("add_fk_%s_%s", td.Name, fk.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)

			// FK depends on referenced table existing
			refCreateID := fmt.Sprintf("create_table_%s", fk.ReferencedTable)
			if _, exists := g.Operations[refCreateID]; exists {
				g.AddDependency(refCreateID, op.ID)
			}

			// FK depends on its own table's columns being added
			for _, col := range fk.Columns {
				addColID := fmt.Sprintf("add_col_%s_%s", td.Name, col)
				if _, exists := g.Operations[addColID]; exists {
					g.AddDependency(addColID, op.ID)
				}
			}
		}
	}

	// Also add FKs for new tables
	for _, table := range diff.AddedTables {
		createTableID := fmt.Sprintf("create_table_%s", table.Name)
		for _, fk := range table.ForeignKeys {
			op := &MigrationOperation{
				Type:     OpAddForeignKey,
				Table:    table.Name,
				Object:   fk.Name,
				Data:     fk,
				ID:       fmt.Sprintf("add_fk_%s_%s", table.Name, fk.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)

			// FK depends on its own table being created
			g.AddDependency(createTableID, op.ID)

			// FK depends on referenced table existing
			refCreateID := fmt.Sprintf("create_table_%s", fk.ReferencedTable)
			if _, exists := g.Operations[refCreateID]; exists {
				g.AddDependency(refCreateID, op.ID)
			}
		}
	}

	// 10. MODIFY PRIMARY KEYS
	for _, td := range diff.Tables {
		if td.PrimaryKeyDiff != nil {
			op := &MigrationOperation{
				Type:     OpModifyPrimaryKey,
				Table:    td.Name,
				Object:   "PRIMARY KEY",
				Data:     td.PrimaryKeyDiff,
				ID:       fmt.Sprintf("mod_pk_%s", td.Name),
				Priority: priority,
			}
			priority++
			g.AddOperation(op)
		}
	}

	// Add cross-table dependencies for DROP TABLE operations
	// Tables with FKs pointing to a table being dropped need special handling
	addDropTableDependencies(g, diff, devSchema)

	// Add CREATE TABLE dependencies based on FK references
	addCreateTableDependencies(g, diff)

	return g
}

// addDropTableDependencies adds dependencies for DROP TABLE operations.
// FKs must be dropped before the tables they reference can be dropped.
func addDropTableDependencies(g *DependencyGraph, diff DiffResult, devSchema *Schema) {
	// Build a map of which tables have FKs pointing to which other tables
	// from the production schema (before changes)
	for _, td := range diff.Tables {
		if td.OnlyInDev {
			continue
		}

		for _, fk := range td.RemovedForeignKeys {
			dropFKID := fmt.Sprintf("drop_fk_%s_%s", td.Name, fk.Name)
			dropRefTableID := fmt.Sprintf("drop_table_%s", fk.ReferencedTable)

			// If the referenced table is being dropped, drop FK first
			if _, exists := g.Operations[dropRefTableID]; exists {
				g.AddDependency(dropFKID, dropRefTableID)
			}
		}
	}
}

// addCreateTableDependencies adds dependencies for CREATE TABLE operations
// based on foreign key references between new tables.
func addCreateTableDependencies(g *DependencyGraph, diff DiffResult) {
	// For added tables, ensure referenced tables are created first
	for _, table := range diff.AddedTables {
		createTableID := fmt.Sprintf("create_table_%s", table.Name)

		for _, fk := range table.ForeignKeys {
			refCreateID := fmt.Sprintf("create_table_%s", fk.ReferencedTable)

			// If both tables are being created, create referenced table first
			if _, exists := g.Operations[refCreateID]; exists {
				if createTableID != refCreateID { // Avoid self-reference
					g.AddDependency(refCreateID, createTableID)
				}
			}
		}
	}
}

// TopologicalSort performs a topological sort on the dependency graph.
// Returns operations in order that respects all dependencies.
func TopologicalSort(g *DependencyGraph) (*OrderingResult, error) {
	result := &OrderingResult{
		Operations: make([]*MigrationOperation, 0, len(g.Operations)),
	}

	// Make a copy of in-degrees for processing
	inDegree := make(map[string]int)
	for id, deg := range g.InDegree {
		inDegree[id] = deg
	}

	// Find all nodes with no incoming edges
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	// Sort initial queue for deterministic output
	sort.Slice(queue, func(i, j int) bool {
		opI := g.Operations[queue[i]]
		opJ := g.Operations[queue[j]]
		if opI.Type != opJ.Type {
			return opI.Type < opJ.Type
		}
		if opI.Priority != opJ.Priority {
			return opI.Priority < opJ.Priority
		}
		return opI.ID < opJ.ID
	})

	processed := 0
	for len(queue) > 0 {
		// Sort queue by operation type and priority for stable ordering
		sort.Slice(queue, func(i, j int) bool {
			opI := g.Operations[queue[i]]
			opJ := g.Operations[queue[j]]
			if opI.Type != opJ.Type {
				return opI.Type < opJ.Type
			}
			if opI.Priority != opJ.Priority {
				return opI.Priority < opJ.Priority
			}
			return opI.ID < opJ.ID
		})

		// Take the first item
		id := queue[0]
		queue = queue[1:]

		op := g.Operations[id]
		result.Operations = append(result.Operations, op)
		processed++

		// Reduce in-degree for all dependents
		for _, depID := range g.Edges[id] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, depID)
			}
		}
	}

	// Check for cycles
	if processed < len(g.Operations) {
		cycles := detectCycles(g, inDegree)
		result.CircularDependencies = cycles

		// Add warning about circular dependencies
		if len(cycles) > 0 {
			for _, cycle := range cycles {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Circular dependency detected: %s", cycle.Message))
			}
		}

		// Add remaining operations (those in cycles) with a warning
		for id, deg := range inDegree {
			if deg > 0 {
				op := g.Operations[id]
				if op != nil {
					result.Operations = append(result.Operations, op)
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Operation %s is part of a dependency cycle and may need manual ordering", op.String()))
				}
			}
		}
	}

	return result, nil
}

// detectCycles finds circular dependencies in the graph.
func detectCycles(g *DependencyGraph, remainingInDegree map[string]int) []CircularDependency {
	var cycles []CircularDependency

	// Find nodes still in the graph (with non-zero in-degree)
	cycleNodes := make(map[string]bool)
	for id, deg := range remainingInDegree {
		if deg > 0 {
			cycleNodes[id] = true
		}
	}

	if len(cycleNodes) == 0 {
		return cycles
	}

	// Use DFS to find cycles
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string, path []string) bool
	dfs = func(node string, path []string) bool {
		if !cycleNodes[node] {
			return false
		}

		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range g.Edges[node] {
			if !cycleNodes[neighbor] {
				continue
			}
			if !visited[neighbor] {
				if dfs(neighbor, path) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle - extract it
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := append(path[cycleStart:], neighbor)
					tables := extractTablesFromCycle(g, cyclePath)
					cycles = append(cycles, CircularDependency{
						Cycle:   cyclePath,
						Tables:  tables,
						Message: fmt.Sprintf("Cycle: %s", strings.Join(cyclePath, " -> ")),
					})
				}
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range cycleNodes {
		if !visited[node] {
			dfs(node, nil)
		}
	}

	return cycles
}

// extractTablesFromCycle extracts unique table names from a cycle.
func extractTablesFromCycle(g *DependencyGraph, cycle []string) []string {
	tableSet := make(map[string]bool)
	for _, id := range cycle {
		if op, exists := g.Operations[id]; exists {
			tableSet[op.Table] = true
		}
	}

	tables := make([]string, 0, len(tableSet))
	for t := range tableSet {
		tables = append(tables, t)
	}
	sort.Strings(tables)
	return tables
}

// OrderedDiffResult contains the diff with operations in dependency order.
type OrderedDiffResult struct {
	// Original diff result
	DiffResult

	// Ordered operations
	OrderedOperations []*MigrationOperation

	// Circular dependencies detected
	CircularDependencies []CircularDependency

	// Warnings about ordering
	Warnings []string
}

// OrderMigrationOperations analyzes a diff result and returns operations
// in the correct dependency order.
func OrderMigrationOperations(diff DiffResult, devSchema *Schema) *OrderedDiffResult {
	result := &OrderedDiffResult{
		DiffResult: diff,
	}

	// Build dependency graph
	graph := BuildDependencyGraph(diff, devSchema)

	// Perform topological sort
	sortResult, _ := TopologicalSort(graph)

	result.OrderedOperations = sortResult.Operations
	result.CircularDependencies = sortResult.CircularDependencies
	result.Warnings = sortResult.Warnings

	return result
}

// GetOrderedAddedTables returns added tables in dependency order.
// Tables that are referenced by other tables' FKs are listed first.
func GetOrderedAddedTables(tables []Table) []Table {
	if len(tables) <= 1 {
		return tables
	}

	// Build a dependency map
	tableMap := make(map[string]Table)
	for _, t := range tables {
		tableMap[t.Name] = t
	}

	// Build adjacency list (table -> tables it depends on)
	deps := make(map[string][]string)
	for _, t := range tables {
		deps[t.Name] = []string{}
		for _, fk := range t.ForeignKeys {
			if _, exists := tableMap[fk.ReferencedTable]; exists {
				deps[t.Name] = append(deps[t.Name], fk.ReferencedTable)
			}
		}
	}

	// Topological sort
	var result []Table
	visited := make(map[string]bool)
	visiting := make(map[string]bool) // For cycle detection

	var visit func(name string) bool
	visit = func(name string) bool {
		if visiting[name] {
			// Cycle detected - break it by returning
			return false
		}
		if visited[name] {
			return true
		}

		visiting[name] = true

		// Visit dependencies first
		for _, dep := range deps[name] {
			visit(dep)
		}

		visiting[name] = false
		visited[name] = true

		if t, exists := tableMap[name]; exists {
			result = append(result, t)
		}
		return true
	}

	// Visit all tables
	for _, t := range tables {
		visit(t.Name)
	}

	return result
}

// GetOrderedRemovedTables returns tables to be dropped in the correct order.
// Tables that reference other tables (via FKs) should be dropped first.
func GetOrderedRemovedTables(tableNames []string, prodSchema *Schema) []string {
	if len(tableNames) <= 1 || prodSchema == nil {
		return tableNames
	}

	// Build a set of tables being removed
	removing := make(map[string]bool)
	for _, name := range tableNames {
		removing[name] = true
	}

	// Build dependency maps:
	// references[A] = [B, C] means A has FKs pointing to B and C
	// referencedBy[B] = [A] means B is referenced by A
	references := make(map[string][]string)
	for _, name := range tableNames {
		references[name] = []string{}
		if table, exists := prodSchema.Tables[name]; exists {
			for _, fk := range table.ForeignKeys {
				if removing[fk.ReferencedTable] {
					references[name] = append(references[name], fk.ReferencedTable)
				}
			}
		}
	}

	// For DROP operations:
	// - If A references B, we must drop A before B
	// - So in topological terms: A comes before B
	// - inDegree[B] increases when A references B (B must wait for A)

	inDegree := make(map[string]int)
	for _, name := range tableNames {
		inDegree[name] = 0
	}

	for name, refs := range references {
		for _, ref := range refs {
			// name references ref, so ref must be dropped after name
			// ref depends on name being dropped first
			inDegree[ref]++
			_ = name // just to clarify name is the one that will decrement ref's count
		}
	}

	// Start with tables that have no incoming edges (no one references them)
	// These are the tables that reference others - they should be dropped first
	var queue []string
	for _, name := range tableNames {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	var result []string
	for len(queue) > 0 {
		// Sort for deterministic order
		sort.Strings(queue)

		name := queue[0]
		queue = queue[1:]
		result = append(result, name)

		// This table is now "dropped", decrement inDegree for tables it references
		for _, ref := range references[name] {
			inDegree[ref]--
			if inDegree[ref] == 0 {
				queue = append(queue, ref)
			}
		}
	}

	// Handle any remaining tables (cycles)
	for _, name := range tableNames {
		found := false
		for _, r := range result {
			if r == name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, name)
		}
	}

	return result
}
