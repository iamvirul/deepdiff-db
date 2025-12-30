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
	Columns  []string `json:"columns"`   // Ordered list of columns in the index
	IsUnique bool     `json:"is_unique"`
}

// Table represents a database table with its columns.
type Table struct {
	Name       string            `json:"name"`
	Columns    map[string]Column `json:"columns"`
	PrimaryKey []string          `json:"primary_key"`
	Indexes    map[string]Index  `json:"indexes,omitempty"`
}

// Schema represents the collection of tables for a database.
type Schema struct {
	Tables map[string]Table `json:"tables"`
}
