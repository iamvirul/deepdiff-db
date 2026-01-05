package errors

import "fmt"

// suggestionMap contains default suggestions for each error code.
// These are general suggestions that apply in most cases.
var suggestionMap = map[ErrorCode][]string{
	// Connection errors
	ErrConnectionFailed: {
		"Verify database host and port are correct",
		"Check network connectivity to database server",
		"Ensure database service is running",
		"Verify firewall rules allow connection",
	},
	ErrConnectionTimeout: {
		"Increase connection timeout in configuration",
		"Check network latency to database server",
		"Verify database is not under heavy load",
	},
	ErrAuthenticationFailed: {
		"Verify database username and password",
		"Check database user permissions",
		"Ensure user has access to the specified database",
	},
	ErrDatabaseNotFound: {
		"Verify database name is correct",
		"Check that the database has been created",
		"Ensure user has permission to access the database",
	},
	ErrQueryFailed: {
		"Check database connection is still active",
		"Verify SQL syntax is correct for the database driver",
		"Check database logs for more details",
	},

	// Schema errors
	ErrSchemaDrift: {
		"Run 'schema-migrate' to generate migration script",
		"Review schema differences in schema_diff.json",
		"Apply schema changes before proceeding with data diff",
	},
	ErrMissingPrimaryKey: {
		"Add a primary key to the table",
		"Use --ignore-tables flag to skip this table",
		"Primary keys are required for accurate data comparison",
	},
	ErrInvalidSchema: {
		"Check database connection and permissions",
		"Verify database driver supports schema introspection",
		"Check database logs for errors",
	},
	ErrColumnMismatch: {
		"Review schema differences in schema_diff.json",
		"Ensure both databases are running compatible versions",
		"Run schema migration to align column definitions",
	},
	ErrSchemaLoadFailed: {
		"Verify database driver is properly initialized",
		"Check user has permission to read schema metadata",
		"Ensure database is accessible",
	},

	// Data errors
	ErrHashingFailed: {
		"Verify table exists and is accessible",
		"Check table has sufficient permissions",
		"Ensure no data corruption in the table",
	},
	ErrDataCorruption: {
		"Run database integrity check (e.g., CHECK TABLE)",
		"Review database logs for errors",
		"Consider restoring from backup",
	},
	ErrConflictDetected: {
		"Review conflicts in conflicts.json",
		"Use 'resolve-conflicts' command to resolve manually",
		"Configure conflict resolution strategy in config file",
	},
	ErrDataComparison: {
		"Ensure both databases are accessible",
		"Verify sufficient memory for comparison",
		"Check for data corruption",
	},

	// Migration errors
	ErrPackGeneration: {
		"Review generation logs for specific errors",
		"Ensure sufficient disk space for pack file",
		"Check write permissions for output directory",
	},
	ErrPackApplication: {
		"Review the failed SQL statement",
		"Check for syntax errors or constraint violations",
		"Verify transaction was properly rolled back",
		"Test migration pack in non-production environment first",
	},
	ErrTransactionFailed: {
		"Check database logs for transaction errors",
		"Verify database has sufficient resources (locks, memory)",
		"Ensure no conflicting transactions are running",
	},
	ErrRollbackFailed: {
		"Database may be in inconsistent state",
		"Review database manually for partial changes",
		"Contact database administrator",
		"Consider restoring from backup",
	},
	ErrMigrationValidation: {
		"Review SQL syntax in migration pack",
		"Test migration in non-production environment",
		"Check for missing tables or columns",
	},

	// Checkpoint errors
	ErrCheckpointRead: {
		"Verify checkpoint file exists and is readable",
		"Check file permissions",
		"Ensure checkpoint file is not corrupted",
	},
	ErrCheckpointWrite: {
		"Verify write permissions for output directory",
		"Check available disk space",
		"Ensure output directory exists",
	},
	ErrCheckpointInvalid: {
		"Checkpoint file may be corrupted",
		"Run without --resume to start fresh",
		"Ensure checkpoint is from compatible version",
	},
	ErrResumeStateMismatch: {
		"Configuration has changed since checkpoint was created",
		"Run without --resume to start fresh",
		"Ensure database and configuration match checkpoint",
	},
	ErrCheckpointExpired: {
		"Checkpoint is too old to resume from safely",
		"Database may have changed since checkpoint",
		"Run without --resume to start fresh",
	},

	// Configuration errors
	ErrConfigInvalid: {
		"Review configuration file for errors",
		"Check YAML syntax is correct",
		"Ensure all required fields are present",
		"Refer to documentation for configuration examples",
	},
	ErrConfigMissing: {
		"Create configuration file at default location",
		"Use --config flag to specify config file path",
		"Refer to documentation for configuration template",
	},
	ErrConfigParse: {
		"Check YAML syntax in configuration file",
		"Ensure proper indentation (use spaces, not tabs)",
		"Validate YAML using online validator",
	},

	// System errors
	ErrOutOfMemory: {
		"Reduce batch size in configuration",
		"Process fewer tables at once",
		"Increase available system memory",
		"Use --ignore-tables to skip large tables",
	},
	ErrDiskFull: {
		"Free up disk space",
		"Clean up old output files",
		"Use different output directory with more space",
	},
	ErrPermissionDenied: {
		"Check file/directory permissions",
		"Run with appropriate user permissions",
		"Verify ownership of files and directories",
	},
	ErrFileNotFound: {
		"Verify file path is correct",
		"Check file exists at specified location",
		"Use absolute path instead of relative path",
	},
}

// GetSuggestions returns contextual suggestions for the given error code and context.
// It combines default suggestions with any context-specific suggestions.
func GetSuggestions(code ErrorCode, context map[string]any) []string {
	// Start with default suggestions for this error code
	suggestions := make([]string, 0)

	if defaults, ok := suggestionMap[code]; ok {
		suggestions = append(suggestions, defaults...)
	}

	// Add context-specific suggestions
	suggestions = append(suggestions, generateContextualSuggestions(code, context)...)

	return suggestions
}

// generateContextualSuggestions creates suggestions based on error context.
func generateContextualSuggestions(code ErrorCode, context map[string]any) []string {
	suggestions := make([]string, 0)

	// Add context-specific suggestions based on error type
	switch code {
	case ErrMissingPrimaryKey:
		if table, ok := context["table"].(string); ok {
			suggestions = append(suggestions,
				fmt.Sprintf("Add primary key to table '%s'", table),
				fmt.Sprintf("Or add '%s' to ignore_tables in config", table))
		}

	case ErrConnectionFailed:
		if host, ok := context["host"].(string); ok {
			if port, ok := context["port"].(int); ok {
				suggestions = append(suggestions,
					fmt.Sprintf("Verify you can reach %s:%d from this machine", host, port))
			}
		}

	case ErrPackApplication:
		if stmt, ok := context["statement"].(string); ok && stmt != "" {
			suggestions = append(suggestions,
				fmt.Sprintf("Review failed SQL: %s", truncate(stmt, 100)))
		}
		if stmtIndex, ok := context["statement_index"].(int); ok {
			suggestions = append(suggestions,
				fmt.Sprintf("Error occurred at statement #%d in the pack", stmtIndex))
		}

	case ErrCheckpointInvalid:
		if path, ok := context["path"].(string); ok {
			suggestions = append(suggestions,
				fmt.Sprintf("Delete checkpoint file: %s", path))
		}
	}

	return suggestions
}

// truncate truncates a string to the specified length, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// AddDefaultSuggestions adds the default suggestions for an error's code to the error.
// This is a convenience method for automatically populating suggestions.
func AddDefaultSuggestions(err *Error) *Error {
	if err == nil {
		return nil
	}

	suggestions := GetSuggestions(err.Code, err.Context)
	for _, s := range suggestions {
		// Only add if not already present
		found := false
		for _, existing := range err.Suggestions {
			if existing == s {
				found = true
				break
			}
		}
		if !found {
			err.Suggestions = append(err.Suggestions, s)
		}
	}

	return err
}
