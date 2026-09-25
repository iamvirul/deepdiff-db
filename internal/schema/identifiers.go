package schema

import "strings"

// CanonicalIdent returns a case- and whitespace-normalized form of a SQL
// identifier, used to match the same logical object across database engines.
//
// Engines disagree on how unquoted identifiers are folded: PostgreSQL lower-cases
// them, while Oracle and DB2 upper-case them. The same logical table is therefore
// stored as "customers" in PostgreSQL but "CUSTOMERS" in Oracle. Canonicalizing to
// lower case lets the diff engine treat them as the same object.
//
// This form is used ONLY for matching objects and for building the engine-agnostic
// row hash. The original identifier is always preserved on the Table/Column structs
// for SQL generation, because quoted identifiers are case-sensitive in most engines
// (notably Oracle): querying "customers" would not find the real table CUSTOMERS.
func CanonicalIdent(ident string) string {
	return strings.ToLower(strings.TrimSpace(ident))
}
