package schema

// Column represents column metadata relevant for diffing.
type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	IsNullable bool   `json:"is_nullable"`
}

// Table represents a database table with its columns.
type Table struct {
	Name    string            `json:"name"`
	Columns map[string]Column `json:"columns"`
}

// Schema represents the collection of tables for a database.
type Schema struct {
	Tables map[string]Table `json:"tables"`
}
