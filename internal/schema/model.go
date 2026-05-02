package schema

// Column represents column metadata relevant for diffing.
type Column struct {
	Name         string  `json:"name"`
	DataType     string  `json:"data_type"`
	IsNullable   bool    `json:"is_nullable"`
	DefaultValue *string `json:"default_value,omitempty"` // Pointer to distinguish between NULL and no default
}

// Index represents a database index on a table.
type Index struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"` // Ordered list of columns in the index
	IsUnique bool     `json:"is_unique"`
}

// ForeignKey represents a foreign key constraint on a table.
type ForeignKey struct {
	Name              string   `json:"name"`                // Constraint name
	Columns           []string `json:"columns"`             // Source columns in this table
	ReferencedTable   string   `json:"referenced_table"`    // Referenced table name
	ReferencedColumns []string `json:"referenced_columns"`  // Referenced columns
	OnDelete          string   `json:"on_delete,omitempty"` // ON DELETE action (CASCADE, SET NULL, etc.)
	OnUpdate          string   `json:"on_update,omitempty"` // ON UPDATE action (CASCADE, SET NULL, etc.)
}

// Table represents a database table with its columns.
type Table struct {
	Name        string                `json:"name"`
	Columns     map[string]Column     `json:"columns"`
	PrimaryKey  []string              `json:"primary_key"`
	Indexes     map[string]Index      `json:"indexes,omitempty"`
	ForeignKeys map[string]ForeignKey `json:"foreign_keys,omitempty"`
}

// RoutineParameter describes a single parameter of a stored routine.
// Mode is one of "IN", "OUT", or "INOUT".
type RoutineParameter struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Mode     string `json:"mode"` // IN | OUT | INOUT
}

// View represents a database view.
// IsMaterialized distinguishes materialized views (e.g. PostgreSQL MATERIALIZED VIEW)
// from standard views.
type View struct {
	Name            string `json:"name"`
	Definition      string `json:"definition"`
	IsMaterialized  bool   `json:"is_materialized"`
}

// Routine represents a stored procedure or function.
// Kind is typically "PROCEDURE" or "FUNCTION".
// ReturnType and Language are omitted when not applicable (e.g. procedures).
type Routine struct {
	Name       string             `json:"name"`
	Kind       string             `json:"kind"`
	Definition string             `json:"definition"`
	Parameters []RoutineParameter `json:"parameters"`
	ReturnType string             `json:"return_type,omitempty"`
	Language   string             `json:"language,omitempty"`
}

// Trigger represents a database trigger bound to a table.
// Timing is one of "BEFORE", "AFTER", or "INSTEAD OF".
// Event is one of "INSERT", "UPDATE", or "DELETE".
type Trigger struct {
	Name       string `json:"name"`
	Table      string `json:"table"`
	Timing     string `json:"timing"`
	Event      string `json:"event"`
	Definition string `json:"definition"`
	ForEachRow bool   `json:"for_each_row"`
}

// Sequence represents a database sequence object.
// Supported on PostgreSQL and Oracle; ignored on drivers that do not have native sequences.
type Sequence struct {
	Name       string `json:"name"`
	StartValue int64  `json:"start_value"`
	Increment  int64  `json:"increment"`
	MinValue   int64  `json:"min_value"`
	MaxValue   int64  `json:"max_value"`
	CacheSize  int64  `json:"cache_size"`
	Cycle      bool   `json:"cycle"`
}

// Schema represents the collection of tables for a database.
type Schema struct {
	Tables    map[string]Table    `json:"tables"`
	Views     map[string]View     `json:"views,omitempty"`
	Routines  map[string]Routine  `json:"routines,omitempty"`
	Triggers  map[string]Trigger  `json:"triggers,omitempty"`
	Sequences map[string]Sequence `json:"sequences,omitempty"`
}
