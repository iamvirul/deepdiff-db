package errors

// ErrorCode represents a unique error identifier for documentation and debugging.
// Error codes follow the format DDDB[category][number] where:
//   - 1xxx: Connection and database access errors
//   - 2xxx: Schema-related errors
//   - 3xxx: Data processing errors
//   - 4xxx: Migration and pack errors
//   - 5xxx: Checkpoint and resume errors
//   - 6xxx: Configuration errors
//   - 7xxx: System resource errors
type ErrorCode string

const (
	// Connection errors (1xxx)

	// ErrConnectionFailed indicates a database connection could not be established.
	ErrConnectionFailed ErrorCode = "DDDB1001"

	// ErrConnectionTimeout indicates a database connection attempt timed out.
	ErrConnectionTimeout ErrorCode = "DDDB1002"

	// ErrAuthenticationFailed indicates database authentication failed.
	ErrAuthenticationFailed ErrorCode = "DDDB1003"

	// ErrDatabaseNotFound indicates the specified database does not exist.
	ErrDatabaseNotFound ErrorCode = "DDDB1004"

	// ErrQueryFailed indicates a database query execution failed.
	ErrQueryFailed ErrorCode = "DDDB1005"

	// Schema errors (2xxx)

	// ErrSchemaDrift indicates schema differences were detected between databases.
	ErrSchemaDrift ErrorCode = "DDDB2001"

	// ErrMissingPrimaryKey indicates a table is missing a required primary key.
	ErrMissingPrimaryKey ErrorCode = "DDDB2002"

	// ErrInvalidSchema indicates the schema is invalid or corrupt.
	ErrInvalidSchema ErrorCode = "DDDB2003"

	// ErrColumnMismatch indicates column definitions don't match between databases.
	ErrColumnMismatch ErrorCode = "DDDB2004"

	// ErrSchemaLoadFailed indicates schema introspection failed.
	ErrSchemaLoadFailed ErrorCode = "DDDB2005"

	// Data errors (3xxx)

	// ErrHashingFailed indicates row hashing operation failed.
	ErrHashingFailed ErrorCode = "DDDB3001"

	// ErrDataCorruption indicates data corruption was detected.
	ErrDataCorruption ErrorCode = "DDDB3002"

	// ErrConflictDetected indicates conflicting data between databases.
	ErrConflictDetected ErrorCode = "DDDB3003"

	// ErrDataComparison indicates data comparison operation failed.
	ErrDataComparison ErrorCode = "DDDB3004"

	// Migration errors (4xxx)

	// ErrPackGeneration indicates migration pack generation failed.
	ErrPackGeneration ErrorCode = "DDDB4001"

	// ErrPackApplication indicates migration pack application failed.
	ErrPackApplication ErrorCode = "DDDB4002"

	// ErrTransactionFailed indicates a database transaction failed.
	ErrTransactionFailed ErrorCode = "DDDB4003"

	// ErrRollbackFailed indicates a transaction rollback failed.
	ErrRollbackFailed ErrorCode = "DDDB4004"

	// ErrMigrationValidation indicates migration validation failed.
	ErrMigrationValidation ErrorCode = "DDDB4005"

	// Checkpoint errors (5xxx)

	// ErrCheckpointRead indicates checkpoint file could not be read.
	ErrCheckpointRead ErrorCode = "DDDB5001"

	// ErrCheckpointWrite indicates checkpoint file could not be written.
	ErrCheckpointWrite ErrorCode = "DDDB5002"

	// ErrCheckpointInvalid indicates checkpoint file is invalid or corrupt.
	ErrCheckpointInvalid ErrorCode = "DDDB5003"

	// ErrResumeStateMismatch indicates resume state doesn't match current operation.
	ErrResumeStateMismatch ErrorCode = "DDDB5004"

	// ErrCheckpointExpired indicates checkpoint is too old to resume from.
	ErrCheckpointExpired ErrorCode = "DDDB5005"

	// Configuration errors (6xxx)

	// ErrConfigInvalid indicates configuration is invalid.
	ErrConfigInvalid ErrorCode = "DDDB6001"

	// ErrConfigMissing indicates required configuration is missing.
	ErrConfigMissing ErrorCode = "DDDB6002"

	// ErrConfigParse indicates configuration file could not be parsed.
	ErrConfigParse ErrorCode = "DDDB6003"

	// System errors (7xxx)

	// ErrOutOfMemory indicates the system ran out of memory.
	ErrOutOfMemory ErrorCode = "DDDB7001"

	// ErrDiskFull indicates the disk is full.
	ErrDiskFull ErrorCode = "DDDB7002"

	// ErrPermissionDenied indicates file or directory permission was denied.
	ErrPermissionDenied ErrorCode = "DDDB7003"

	// ErrFileNotFound indicates a required file was not found.
	ErrFileNotFound ErrorCode = "DDDB7004"
)

// Category returns the high-level category of the error based on its code.
func (e ErrorCode) Category() string {
	if len(e) < 6 {
		return "Unknown"
	}

	switch e[4] {
	case '1':
		return "Connection"
	case '2':
		return "Schema"
	case '3':
		return "Data"
	case '4':
		return "Migration"
	case '5':
		return "Checkpoint"
	case '6':
		return "Configuration"
	case '7':
		return "System"
	default:
		return "Unknown"
	}
}

// String returns the string representation of the error code.
func (e ErrorCode) String() string {
	return string(e)
}

// IsRetryable returns true if errors with this code are potentially retryable.
func (e ErrorCode) IsRetryable() bool {
	retryable := map[ErrorCode]bool{
		ErrConnectionTimeout: true,
		ErrConnectionFailed:  true,
		ErrQueryFailed:       true,
		ErrTransactionFailed: true,
	}
	return retryable[e]
}
