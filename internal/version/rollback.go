package version

import (
	"fmt"

	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// GenerateRollbackSQL generates SQL that undoes the changes captured in c.
//
// It computes the inverse diff (dev → prod) using the stored schemas and passes
// it to schema.GenerateMigration. The returned string is a complete, driver-aware
// migration script with the same safety defaults as schema-migrate.
//
// driver overrides c.Driver when non-empty.
func GenerateRollbackSQL(c *Commit, driver string) (string, error) {
	if c.ProdSchema == nil || c.DevSchema == nil {
		return "", fmt.Errorf("commit %s does not contain schema snapshots; rollback requires re-committing with the current version", shortHash(c.Hash))
	}
	if driver == "" {
		driver = c.Driver
	}
	if driver == "" {
		return "", fmt.Errorf("driver is required for rollback SQL generation (use --driver flag or ensure the commit has a driver set)")
	}

	// Inverse diff: bring dev back to prod state.
	rollbackDiff := schema.DiffSchemas(c.DevSchema, c.ProdSchema)

	sql, err := schema.GenerateMigrationWithSchemas(rollbackDiff, driver, nil, c.DevSchema)
	if err != nil {
		return "", fmt.Errorf("generate rollback migration: %w", err)
	}
	return sql, nil
}

// InterVersionDiff computes a schema.DiffResult describing how the dev schema
// evolved from commit a to commit b.  This is used by "version diff <h1> <h2>".
func InterVersionDiff(a, b *Commit) (schema.DiffResult, error) {
	if a.DevSchema == nil {
		return schema.DiffResult{}, fmt.Errorf("commit %s is missing DevSchema snapshot", shortHash(a.Hash))
	}
	if b.DevSchema == nil {
		return schema.DiffResult{}, fmt.Errorf("commit %s is missing DevSchema snapshot", shortHash(b.Hash))
	}
	return schema.DiffSchemas(a.DevSchema, b.DevSchema), nil
}

