package content

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// ApplyPack executes a SQL migration pack file against the target database.
// ApplyPack reads the SQL migration pack at packPath and either validates its statements or applies them to db.
// 
// If dryRun is true, each non-empty statement is prepared against db to validate syntax; the first preparation
// error is returned with the 1-based statement index. If dryRun is false, all non-empty statements are executed
// in a single transaction; any execution error is returned with the 1-based statement index and the transaction
// is rolled back. Returns an error if the pack file cannot be read or is empty, or if beginning or committing
// the transaction fails.
func ApplyPack(ctx context.Context, db *sql.DB, packPath string, dryRun bool) error {
	data, err := os.ReadFile(packPath)
	if err != nil {
		return fmt.Errorf("read pack file: %w", err)
	}

	sqlText := strings.TrimSpace(string(data))
	if sqlText == "" {
		return fmt.Errorf("pack file is empty")
	}

	if dryRun {
		// Validate SQL syntax by preparing statements
		statements := SplitStatements(sqlText)
		for i, stmt := range statements {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := db.PrepareContext(ctx, stmt); err != nil {
				return fmt.Errorf("dry-run validation failed at statement %d: %w", i+1, err)
			}
		}
		return nil
	}

	// Execute in a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore rollback errors (transaction may already be committed)
	}()

	statements := SplitStatements(sqlText)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute statement %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// splitStatements splits sqlText into individual SQL statements separated by semicolons.
// It preserves semicolons that appear inside single quotes, double quotes, or backticks
// and respects backslash-escaped characters when determining statement boundaries.
// The returned slice contains trimmed, non-empty statements; a final statement is included
// even if the input does not end with a semicolon.
func splitStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	escapeNext := false
	quoteChar := byte(0)

	for i := 0; i < len(sqlText); i++ {
		char := sqlText[i]

		if escapeNext {
			current.WriteByte(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			current.WriteByte(char)
			continue
		}

		if !inString {
			if char == '\'' || char == '"' || char == '`' {
				inString = true
				quoteChar = char
				current.WriteByte(char)
			} else if char == ';' {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
			} else {
				current.WriteByte(char)
			}
		} else {
			current.WriteByte(char)
			if char == quoteChar {
				inString = false
				quoteChar = 0
			}
		}
	}

	// Handle final statement without trailing semicolon
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// SplitStatements splits SQL text by semicolons, handling edge cases.
// SplitStatements splits sqlText into individual SQL statements.
// It treats semicolons that occur outside string literals as statement terminators,
// preserves quoted strings using single quotes ('), double quotes ("), and backticks (`),
// respects backslash escapes inside strings, trims surrounding whitespace from each statement,
// and omits empty statements from the result.
func SplitStatements(sqlText string) []string {
	return splitStatements(sqlText)
}