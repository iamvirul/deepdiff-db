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

// HasChanges reports whether any table has additions/removals/updates.
func (d DataDiff) HasChanges() bool {
	for _, t := range d.Tables {
		if len(t.Added) > 0 || len(t.Removed) > 0 || len(t.Updated) > 0 {
			return true
		}
	}
	return false
}

// DiffTableHashes compares two hash maps keyed by primary key composite.
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

// BuildDataDiff produces diffs for all shared tables (schema drift should be checked separately).
func BuildDataDiff(prodSchema, devSchema *schema.Schema, prodHashes, devHashes map[string]map[string]string) DataDiff {
	diff := DataDiff{}
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; !ok {
			continue
		}

		pHashes := prodHashes[name]
		dHashes := devHashes[name]

		td := DiffTableHashes(name, pHashes, dHashes)
		diff.Tables = append(diff.Tables, td)
	}
	return diff
}
