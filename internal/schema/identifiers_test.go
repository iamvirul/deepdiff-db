package schema

import "testing"

func TestCanonicalIdent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"postgres lower", "customers", "customers"},
		{"oracle upper", "CUSTOMERS", "customers"},
		{"mixed case", "Customers", "customers"},
		{"surrounding whitespace", "  customers  ", "customers"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanonicalIdent(tc.in); got != tc.want {
				t.Errorf("CanonicalIdent(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestCanonicalIdent_CrossEngineMatch documents the core invariant the diff
// engine relies on: identifiers that a migration folds to a different case
// (PostgreSQL lower vs Oracle upper) canonicalize to the same key.
func TestCanonicalIdent_CrossEngineMatch(t *testing.T) {
	if CanonicalIdent("customers") != CanonicalIdent("CUSTOMERS") {
		t.Error("expected PostgreSQL and Oracle folded names to canonicalize equal")
	}
}
