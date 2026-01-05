package logger

// Common field names for consistent structured logging across the application.
// These constants ensure uniform field naming in log entries.
const (
	// FieldTable is the field name for database table names.
	FieldTable = "table"

	// FieldDriver is the field name for database driver type (mysql, postgres, sqlite).
	FieldDriver = "driver"

	// FieldDatabase is the field name for database name.
	FieldDatabase = "database"

	// FieldOperation is the field name for operation type.
	FieldOperation = "operation"

	// FieldDuration is the field name for operation duration in milliseconds.
	FieldDuration = "duration_ms"

	// FieldRowCount is the field name for row count.
	FieldRowCount = "row_count"

	// FieldError is the field name for error messages.
	FieldError = "error"

	// FieldPhase is the field name for operation phase.
	FieldPhase = "phase"

	// FieldHost is the field name for database host.
	FieldHost = "host"

	// FieldPort is the field name for database port.
	FieldPort = "port"

	// FieldPath is the field name for file paths.
	FieldPath = "path"

	// FieldStatementCount is the field name for SQL statement count.
	FieldStatementCount = "statement_count"

	// FieldSize is the field name for file or data size.
	FieldSize = "size_bytes"
)
