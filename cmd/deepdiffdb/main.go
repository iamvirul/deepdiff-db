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

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("deepdiffdb: %v", err)
	}
}

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
	case "diff":
		return runFullDiff(args[1:])
	case "gen-pack":
		return runGenPack(args[1:])
	case "apply":
		return runApply(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

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

	packPath, err := content.GeneratePack(ctx, cfg.Prod.Driver, devDB, devSchema, dataDiff, ignoreColumn, cfg.Output.Dir)
	if err != nil {
		return fmt.Errorf("generate pack: %w", err)
	}

	if err := content.WriteReportsWithInfo(dataDiff, conflicts, cfg.Output.Dir, "OK", tablesScanned, packPath); err != nil {
		return fmt.Errorf("write content diff: %w", err)
	}

	fmt.Println("Schema OK. Data diff complete.")
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

func printNotImplemented(cmd string) {
	fmt.Printf("%s command is not implemented yet. Coming soon.\n", cmd)
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`DeepDiff DB (Go CLI)

Usage:
  %[1]s <command> [flags]

Commands:
  check         Validate configuration and show quick summary
  schema-diff   Detect schema drift
  diff          Full diff: schema + data
  gen-pack      Generate SQL migration pack
  apply         Apply migration pack

Use "%[1]s <command> -h" for flags specific to that command.
`, exe)
}
