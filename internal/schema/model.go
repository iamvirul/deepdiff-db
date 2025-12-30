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

// Schema represents the collection of tables for a database.
type Schema struct {
	Tables map[string]Table `json:"tables"`
}
