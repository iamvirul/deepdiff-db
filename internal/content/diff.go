package content

import "github.com/iamvirul/deepdiff-db/internal/schema"

// TableDataDiff captures row-level differences for one table.
type TableDataDiff struct {
	Table   string   `json:"table"`
	Added   []string `json:"added_keys,omitempty"`
	Removed []string `json:"removed_keys,omitempty"`
	Updated []string `json:"updated_keys,omitempty"`
}

// DataDiff aggregates all table diffs.
type DataDiff struct {
	Tables []TableDataDiff `json:"tables"`
}

// Conflict represents a row that exists in both prod and dev but differs.
type Conflict struct {
	Table    string `json:"table"`
	Key      string `json:"key"`
	ProdHash string `json:"prod_hash"`
	DevHash  string `json:"dev_hash"`
}

// Conflicts aggregates all conflicts.
type Conflicts struct {
	Conflicts []Conflict `json:"conflicts"`
}

// HasConflicts reports whether any conflicts exist.
func (c Conflicts) HasConflicts() bool {
	return len(c.Conflicts) > 0
}

// HasChanges reports whether any table has additions/removals/updates.
func (d DataDiff) HasChanges() bool {
	for _, t := range d.Tables {
		if len(t.Added) > 0 || len(t.Removed) > 0 || len(t.Updated) > 0 {
			return true
		}
	}
	return false
}

// DiffTableHashes computes per-key differences between prod and dev hash maps for a table.
// It returns a TableDataDiff with keys present only in prod listed in Removed, keys present
// only in dev listed in Added, and keys present in both with differing hash values listed in Updated.
func DiffTableHashes(table string, prod, dev map[string]string) TableDataDiff {
	td := TableDataDiff{Table: table}

	for k, prodHash := range prod {
		if devHash, ok := dev[k]; !ok {
			td.Removed = append(td.Removed, k)
		} else if devHash != prodHash {
			td.Updated = append(td.Updated, k)
		}
	}
	for k := range dev {
		if _, ok := prod[k]; !ok {
			td.Added = append(td.Added, k)
		}
	}

	return td
}

// BuildDataDiff builds per-table data diffs for tables present in both prodSchema and devSchema.
// It compares the provided per-table row-hash maps and records added, removed, and updated keys for each shared table.
// It also collects conflicts for rows that exist in both prod and dev but have different hashes.
// The returned DataDiff aggregates TableDataDiff entries for each shared table; the returned Conflicts lists each differing row as a Conflict (table, key, prod hash, dev hash).
func BuildDataDiff(prodSchema, devSchema *schema.Schema, prodHashes, devHashes map[string]map[string]string) (DataDiff, Conflicts) {
	diff := DataDiff{}
	conflicts := Conflicts{}
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; !ok {
			continue
		}

		pHashes := prodHashes[name]
		dHashes := devHashes[name]

		td := DiffTableHashes(name, pHashes, dHashes)
		diff.Tables = append(diff.Tables, td)

		// Detect conflicts (rows that exist in both but differ)
		for k, prodHash := range pHashes {
			if devHash, ok := dHashes[k]; ok && devHash != prodHash {
				conflicts.Conflicts = append(conflicts.Conflicts, Conflict{
					Table:    name,
					Key:      k,
					ProdHash: prodHash,
					DevHash:  devHash,
				})
			}
		}
	}
	return diff, conflicts
}
