package content

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// ApplyPack executes a SQL migration pack file against the target database.
// If dryRun is true, it validates the SQL but does not execute it.
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
	defer tx.Rollback()

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
// Exported for testing purposes.
func SplitStatements(sqlText string) []string {
	return splitStatements(sqlText)
}
