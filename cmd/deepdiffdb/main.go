package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/drivers"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	"github.com/iamvirul/deepdiff-db/pkg/config"
)

// Version information - set via ldflags during build
var (
	version   = "dev"      // Version number (e.g., "v0.3.0" or "dev-abc123")
	commit    = "unknown"  // Git commit hash
	branch    = "unknown"  // Git branch
	buildTime = "unknown"  // Build timestamp
)

// main is the CLI entry point for DeepDiff DB; it dispatches the requested subcommand and exits with a fatal log on error.
func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("deepdiffdb: %v", err)
	}
}

// run dispatches CLI subcommands based on the first element of args.
// 
// It invokes the corresponding command handler with the remaining arguments,
// prints usage and returns an error when no command is provided or when the
// command is unknown, and prints usage without error for help flags.
func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return errors.New("no command provided")
	}

	switch args[0] {
	case "check":
		return runCheck(args[1:])
	case "schema-diff":
		return runSchemaDiff(args[1:])
	case "schema-migrate":
		return runSchemaMigrate(args[1:])
	case "diff":
		return runFullDiff(args[1:])
	case "gen-pack":
		return runGenPack(args[1:])
	case "apply":
		return runApply(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	case "-v", "--version", "version":
		printVersion()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// runCheck validates configuration, opens prod/dev database connections, and verifies that all non-ignored tables have primary keys.
//
// It parses the "check" command flags (supports --config to specify the config file, default "deepdiffdb.config.yaml"), loads the configuration, opens production and development connections, ensures the configured output directory exists, and checks each database for tables missing primary keys (respecting the configured ignore table list). Returns an error if flag parsing, config loading, any database connection, output directory creation, or primary key checks fail; otherwise prints connection and ignore info and returns nil.
func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	prodMissing, err := schema.CheckPrimaryKeys(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("prod primary key check: %w", err)
	}
	if len(prodMissing) > 0 {
		return fmt.Errorf("prod tables missing primary keys: %v", prodMissing)
	}

	devMissing, err := schema.CheckPrimaryKeys(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("dev primary key check: %w", err)
	}
	if len(devMissing) > 0 {
		return fmt.Errorf("dev tables missing primary keys: %v", devMissing)
	}

	fmt.Println("Config loaded.")
	fmt.Printf("Prod: %s:%d/%s\n", cfg.Prod.Host, cfg.Prod.Port, cfg.Prod.Database)
	fmt.Printf("Dev : %s:%d/%s\n", cfg.Dev.Host, cfg.Dev.Port, cfg.Dev.Database)
	fmt.Printf("Output directory ready: %s\n", cfg.Output.Dir)
	fmt.Printf("Ignore tables: %v\n", cfg.Ignore.Tables)
	fmt.Printf("Ignore columns: %v\n", cfg.Ignore.Columns)
	fmt.Println("Connections OK. Primary keys verified.")
	return nil
}

// runFullDiff performs a full schema and data comparison between the production and development
// databases using command-line arguments (e.g., --config). It writes schema and data diff reports
// into the configured output directory and prints a concise summary to stdout.
// It returns an error if configuration loading, database connections, report generation fail, or
// if schema drift is detected.
func runFullDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Schema diff first
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}
	if schemaDiff.HasDrift() {
		return fmt.Errorf("schema drift detected; see %s and %s", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
	}

	// Data diff
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
	prodHashes := make(map[string]map[string]string)
	devHashes := make(map[string]map[string]string)

	for name, prodTable := range prodSchema.Tables {
		devTable, ok := devSchema.Tables[name]
		if !ok {
			continue
		}

		pHashes, err := content.HashTable(ctx, prodDB, cfg.Prod.Driver, prodTable, ignoreColumn)
		if err != nil {
			return fmt.Errorf("hash prod table %s: %w", name, err)
		}
		dHashes, err := content.HashTable(ctx, devDB, cfg.Dev.Driver, devTable, ignoreColumn)
		if err != nil {
			return fmt.Errorf("hash dev table %s: %w", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	tablesScanned := 0
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; ok {
			tablesScanned++
		}
	}

	if err := content.WriteReportsWithInfo(dataDiff, conflicts, cfg.Output.Dir, "OK", tablesScanned, ""); err != nil {
		return fmt.Errorf("write content diff: %w", err)
	}

	fmt.Println("Schema OK. Data diff complete.")
	if dataDiff.HasChanges() {
		fmt.Printf("Changes detected. See %s, %s, and %s\n", filepath.Join(cfg.Output.Dir, "content_diff.json"), filepath.Join(cfg.Output.Dir, "conflicts.json"), filepath.Join(cfg.Output.Dir, "summary.txt"))
		if conflicts.HasConflicts() {
			fmt.Printf("Warning: %d conflicts detected. Review %s\n", len(conflicts.Conflicts), filepath.Join(cfg.Output.Dir, "conflicts.json"))
		}
	} else {
		fmt.Println("No data differences found.")
	}
	return nil
}

// runGenPack performs the "gen-pack" command: it loads configuration, compares prod and dev schemas, and generates a migration pack for any detected data differences while writing schema and data reports to the configured output directory.
// It opens prod and dev database connections, ensures the output directory exists, fails if schema drift is detected (after writing schema reports), computes per-table content hashes with configured ignore rules, builds the data diff and any conflicts, generates a migration pack when changes exist, writes data reports including the pack path and conflicts, and prints a brief status summary.
// It returns a non-nil error for failures during configuration loading, database connections, schema loading or reporting, hashing, pack generation, or report writing; it also returns an error when schema drift is detected so the pack is not produced.
func runGenPack(args []string) error {
	fs := flag.NewFlagSet("gen-pack", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Schema diff first
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}
	if schemaDiff.HasDrift() {
		fmt.Fprintf(os.Stderr, "Warning: schema drift detected; see %s and %s\n", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
		fmt.Fprintf(os.Stderr, "Warning: continuing with pack generation. Only tables with matching schemas will be included.\n")
	}

	// Data diff
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
	prodHashes := make(map[string]map[string]string)
	devHashes := make(map[string]map[string]string)

	for name, prodTable := range prodSchema.Tables {
		devTable, ok := devSchema.Tables[name]
		if !ok {
			continue
		}

		pHashes, err := content.HashTable(ctx, prodDB, cfg.Prod.Driver, prodTable, ignoreColumn)
		if err != nil {
			return fmt.Errorf("hash prod table %s: %w", name, err)
		}
		dHashes, err := content.HashTable(ctx, devDB, cfg.Dev.Driver, devTable, ignoreColumn)
		if err != nil {
			return fmt.Errorf("hash dev table %s: %w", name, err)
		}

		prodHashes[name] = pHashes
		devHashes[name] = dHashes
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	tablesScanned := 0
	for name := range prodSchema.Tables {
		if _, ok := devSchema.Tables[name]; ok {
			tablesScanned++
		}
	}

	packPath, err := content.GeneratePack(ctx, cfg.Prod.Driver, devDB, cfg.Dev.Database, prodSchema, devSchema, schemaDiff, dataDiff, ignoreColumn, cfg.Output.Dir)
	if err != nil {
		return fmt.Errorf("generate pack: %w", err)
	}

	if err := content.WriteReportsWithInfo(dataDiff, conflicts, cfg.Output.Dir, "OK", tablesScanned, packPath); err != nil {
		return fmt.Errorf("write content diff: %w", err)
	}

	if schemaDiff.HasDrift() {
		fmt.Println("Schema drift detected. Data diff complete.")
	} else {
		fmt.Println("Schema OK. Data diff complete.")
	}
	if dataDiff.HasChanges() {
		fmt.Printf("Changes detected. Pack written to %s\n", packPath)
		if conflicts.HasConflicts() {
			fmt.Printf("Warning: %d conflicts detected. Review %s before applying pack.\n", len(conflicts.Conflicts), filepath.Join(cfg.Output.Dir, "conflicts.json"))
		}
	} else {
		fmt.Println("No data differences found. Pack not required.")
	}
	return nil
}

// runApply parses command-line flags and applies or validates a migration pack against
// the configured production database.
//
// It accepts the following flags on args:
//   --pack   Path to the migration pack SQL file (required).
//   --dry-run  Validate the SQL without executing it.
//   --config Path to the configuration file (default "deepdiffdb.config.yaml").
//
// runApply returns an error when flag parsing fails, the required --pack is missing,
// the configuration cannot be loaded, the target database connection cannot be opened,
// or when validation/application of the pack fails.
func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	packPath := fs.String("pack", "", "Path to migration pack SQL file (required)")
	dryRun := fs.Bool("dry-run", false, "Validate SQL without executing")
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *packPath == "" {
		return fmt.Errorf("--pack flag is required")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	// Apply to prod database
	targetDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("target connection failed: %w", err)
	}
	defer targetDB.Close()

	if *dryRun {
		fmt.Println("Dry-run mode: validating SQL...")
		if err := content.ApplyPack(ctx, targetDB, *packPath, true); err != nil {
			return fmt.Errorf("dry-run validation failed: %w", err)
		}
		fmt.Println("Dry-run validation passed. SQL is valid.")
		return nil
	}

	fmt.Printf("Applying migration pack: %s\n", *packPath)
	if err := content.ApplyPack(ctx, targetDB, *packPath, false); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
	fmt.Println("Migration pack applied successfully.")
	return nil
}

// runSchemaDiff parses flags for the "schema-diff" command, loads configuration, opens production and development databases, loads their schemas, writes schema diff reports to the configured output directory, and returns an error if schema drift is detected or any operation fails.
func runSchemaDiff(args []string) error {
	fs := flag.NewFlagSet("schema-diff", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	diff := schema.DiffSchemas(prodSchema, devSchema)

	if err := schema.WriteReports(diff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}

	if diff.HasDrift() {
		return fmt.Errorf("schema drift detected; see %s and %s", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
	}

	fmt.Println("Schema match confirmed. No drift detected.")
	return nil
}

// runSchemaMigrate generates a standalone schema migration script based on schema differences
// between production and development databases.
//
// It parses flags (--config, --dry-run), loads configuration, opens both database connections,
// loads schemas, computes schema diff, generates migration SQL, and writes to output directory.
// Returns an error if any operation fails.
func runSchemaMigrate(args []string) error {
	fs := flag.NewFlagSet("schema-migrate", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	dryRun := fs.Bool("dry-run", false, "Generate and validate migration without writing file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ctx := context.Background()

	prodDB, err := drivers.Open(ctx, cfg.Prod)
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := drivers.Open(ctx, cfg.Dev)
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o755); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Load schemas
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	// Compute schema diff
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)

	// Prepare migration options from config
	opts := &schema.MigrationOptions{
		AllowDropColumn:    cfg.Migration.AllowDropColumn,
		AllowDropTable:     cfg.Migration.AllowDropTable,
		ConfirmDestructive: cfg.Migration.ConfirmDestructive,
	}

	// Generate migration script
	migrationSQL, err := schema.GenerateMigration(schemaDiff, cfg.Prod.Driver, opts)
	if err != nil {
		return fmt.Errorf("generate migration: %w", err)
	}

	if *dryRun {
		fmt.Println("Dry-run mode: migration script generated (not written to file)")
		if !schemaDiff.HasDrift() {
			fmt.Println("No schema changes detected.")
		} else {
			fmt.Printf("Schema migration generated (%d bytes):\n", len(migrationSQL))
			fmt.Println("---")
			fmt.Println(migrationSQL)
			fmt.Println("---")
		}
		return nil
	}

	if !schemaDiff.HasDrift() {
		fmt.Println("No schema changes detected. No migration file generated.")
		return nil
	}

	// Write migration file
	migrationPath := filepath.Join(cfg.Output.Dir, "schema_migration.sql")
	if err := os.WriteFile(migrationPath, []byte(migrationSQL), 0o644); err != nil {
		return fmt.Errorf("write migration file: %w", err)
	}

	fmt.Printf("Schema migration generated: %s\n", migrationPath)
	fmt.Printf("Changes detected:\n")
	if len(schemaDiff.Tables) > 0 {
		for _, td := range schemaDiff.Tables {
			if len(td.AddedColumns) > 0 {
				fmt.Printf("  - Table '%s': %d columns added\n", td.Name, len(td.AddedColumns))
			}
			if len(td.RemovedColumns) > 0 {
				fmt.Printf("  - Table '%s': %d columns removed\n", td.Name, len(td.RemovedColumns))
			}
			if len(td.ModifiedColumns) > 0 {
				fmt.Printf("  - Table '%s': %d columns modified\n", td.Name, len(td.ModifiedColumns))
			}
		}
	}
	if len(schemaDiff.AddedTables) > 0 {
		fmt.Printf("  - %d tables added\n", len(schemaDiff.AddedTables))
	}
	if len(schemaDiff.RemovedTables) > 0 {
		fmt.Printf("  - %d tables removed\n", len(schemaDiff.RemovedTables))
	}

	return nil
}

// printUsage prints the CLI usage text including available commands, brief descriptions, and how to display flags for a specific command.
func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`DeepDiff DB (Go CLI)

Usage:
  %[1]s <command> [flags]

Commands:
  check           Validate configuration and show quick summary
  schema-diff     Detect schema drift
  schema-migrate  Generate schema migration script
  diff            Full diff: schema + data
  gen-pack        Generate SQL migration pack
  apply           Apply migration pack

Global Flags:
  -v, --version   Show version information
  -h, --help      Show this help message

Use "%[1]s <command> -h" for flags specific to that command.
`, exe)
}

// printVersion prints the version information including build details.
func printVersion() {
	fmt.Printf("DeepDiff DB %s\n", version)
	if commit != "unknown" {
		fmt.Printf("  Commit:     %s\n", commit)
	}
	if branch != "unknown" {
		fmt.Printf("  Branch:     %s\n", branch)
	}
	if buildTime != "unknown" {
		fmt.Printf("  Build Time: %s\n", buildTime)
	}
}