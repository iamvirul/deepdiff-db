// Package version implements a Git-like versioning system for database diffs.
// Each commit stores a snapshot of the schema and data diff between two databases,
// along with metadata (author, message, timestamp) and enough schema context to
// generate rollback SQL.
package version

import (
	"time"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/schema"
)

// RepoDirName is the directory created by "version init" inside the working directory.
const RepoDirName = ".deepdiffdb"

// objectsDirName is the subdirectory that stores one JSON file per commit.
const objectsDirName = "objects"

// headFileName holds the hash of the most recent commit (empty string when no commits yet).
const headFileName = "HEAD"

// Commit is a versioned snapshot of a database diff.
//
// ProdSchema and DevSchema are stored verbatim so that rollback SQL can be
// generated without requiring a live database connection.
type Commit struct {
	// Hash is the SHA-256 hex digest of the commit's canonical content.
	Hash string `json:"hash"`

	// Parent is the hash of the preceding commit, empty for the initial commit.
	Parent string `json:"parent,omitempty"`

	Timestamp time.Time `json:"timestamp"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`

	// Driver is the prod database driver (mysql/postgres/sqlite/mssql/oracle).
	// Used as the default when generating migration/rollback SQL.
	Driver string `json:"driver"`

	SchemaDiff schema.DiffResult `json:"schema_diff"`
	DataDiff   content.DataDiff  `json:"data_diff"`

	// ProdSchema and DevSchema are kept for rollback and inter-version diff generation.
	ProdSchema *schema.Schema `json:"prod_schema,omitempty"`
	DevSchema  *schema.Schema `json:"dev_schema,omitempty"`
}
