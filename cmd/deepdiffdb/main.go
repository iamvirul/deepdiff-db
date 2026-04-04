package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/iamvirul/deepdiff-db/internal/checkpoint"
	"github.com/iamvirul/deepdiff-db/internal/cli"
	"github.com/iamvirul/deepdiff-db/internal/content"
	"github.com/iamvirul/deepdiff-db/internal/content/resolve"
	"github.com/iamvirul/deepdiff-db/internal/drivers"
	htmlreport "github.com/iamvirul/deepdiff-db/internal/report/html"
	"github.com/iamvirul/deepdiff-db/internal/schema"
	vcs "github.com/iamvirul/deepdiff-db/internal/version"
	"github.com/iamvirul/deepdiff-db/pkg/config"
	"github.com/iamvirul/deepdiff-db/pkg/logger"
	"github.com/iamvirul/deepdiff-db/pkg/progress"
)

// Version information - set via ldflags during build
var (
	version   = "dev"     // Version number (e.g., "v0.3.0" or "dev-abc123")
	commit    = "unknown" // Git commit hash
	branch    = "unknown" // Git branch
	buildTime = "unknown" // Build timestamp
)

// initializeLogger creates and configures a logger based on command-line flags.
// It handles log level parsing, file output setup, and format selection.
// Returns a configured logger ready for use throughout the application.
func initializeLogger(verbose bool, logFile string, logLevelStr string, logFormat string) (*logger.Logger, io.Closer, error) {
	// Parse log level
	level := logger.ParseLevel(logLevelStr)

	// If verbose mode, use debug level
	if verbose {
		level = slog.LevelDebug
	}

	// Validate log format
	if logFormat == "" {
		logFormat = "text"
	}
	if logFormat != "text" && logFormat != "json" {
		return nil, nil, fmt.Errorf("invalid log format: %s (must be 'text' or 'json')", logFormat)
	}

	// Open log file if specified
	var fileOutput io.Writer
	var fileCloser io.Closer
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open log file: %w", err)
		}
		fileOutput = f
		fileCloser = f
	}

	// Create logger configuration
	cfg := logger.Config{
		Level:         level,
		Format:        logFormat,
		Output:        os.Stdout,
		FileOutput:    fileOutput,
		WithSource:    verbose, // Include source location in verbose mode
		EnableMetrics: true,    // Always enable metrics for performance tracking
	}

	log := logger.New(cfg)
	return log, fileCloser, nil
}

// openDatabaseWithSpinner opens a database connection with optional spinner for progress indication.
// This is used for operations where connection time is unknown.
func openDatabaseWithSpinner(ctx context.Context, cfg config.DBConfig, dbName string) (*sql.DB, error) {
	progressMgr := progress.FromContext(ctx)

	var spinner *progress.Bar
	if progressMgr != nil && progressMgr.IsEnabled() {
		spinner = progressMgr.StartSpinner(ctx, fmt.Sprintf("Connecting to %s", dbName))
		defer func() {
			_ = spinner.Finish() // Ignore error - spinner is finishing anyway
		}()
	}

	db, err := drivers.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// main is the CLI entry point for DeepDiff DB; it dispatches the requested subcommand and exits with a fatal log on error.
func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("deepdiffdb: %v", err)
	}
}

// run dispatches CLI subcommands based on the first element of args.
// hashTablesParallel hashes all tables in parallel, bounded by maxParallel goroutines.
//
// It spawns one goroutine per table and uses a semaphore to cap concurrency at
// maxParallel. Results are written into the returned map under a mutex so callers
// receive a fully-populated map[tableName]map[pkKey]hash on success.
//
// If any goroutine fails, the errgroup cancels the shared context and returns the
// first error encountered. Partial results are not returned on error.
func hashTablesParallel(
	ctx context.Context,
	tables map[string]schema.Table,
	db *sql.DB,
	driver string,
	ignoreFn func(string, string) bool,
	batchSize, maxParallel int,
) (map[string]map[string]string, error) {
	if maxParallel <= 0 {
		maxParallel = 1
	}

	results := make(map[string]map[string]string, len(tables))
	var mu sync.Mutex

	sem := semaphore.NewWeighted(int64(maxParallel))
	eg, egCtx := errgroup.WithContext(ctx)

	for name, tbl := range tables {
		name, tbl := name, tbl // capture loop vars
		eg.Go(func() error {
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err // context cancelled
			}
			defer sem.Release(1)

			hashes, err := content.HashTable(egCtx, db, driver, tbl, ignoreFn, batchSize)
			if err != nil {
				return fmt.Errorf("hash table %s: %w", name, err)
			}

			mu.Lock()
			results[name] = hashes
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// resolveParallel returns maxParallel from flagVal when > 0, otherwise from cfg.
func resolveParallel(flagVal int, cfg int) int {
	if flagVal > 0 {
		return flagVal
	}
	return cfg
}

// resolveBatchSize returns batchSize from flagVal when > 0, otherwise from cfg.
func resolveBatchSize(flagVal int, cfg int) int {
	if flagVal > 0 {
		return flagVal
	}
	return cfg
}

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
	case "resolve-conflicts":
		return runResolveConflicts(args[1:])
	case "version":
		return runVersion(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	case "-v", "--version":
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
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting configuration check")

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	ctx := logger.ToContext(context.Background(), log)

	log.Info("connecting to production database",
		logger.FieldDriver, cfg.Prod.Driver,
		logger.FieldHost, cfg.Prod.Host,
		logger.FieldPort, cfg.Prod.Port,
		logger.FieldDatabase, cfg.Prod.Database)

	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	log.Info("connecting to development database",
		logger.FieldDriver, cfg.Dev.Driver,
		logger.FieldHost, cfg.Dev.Host,
		logger.FieldPort, cfg.Dev.Port,
		logger.FieldDatabase, cfg.Dev.Database)

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	log.Debug("output directory ready", logger.FieldPath, cfg.Output.Dir)

	log.Info("checking primary keys on production database")
	prodMissing, err := schema.CheckPrimaryKeys(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("prod primary key check: %w", err)
	}
	if len(prodMissing) > 0 {
		return fmt.Errorf("prod tables missing primary keys: %v", prodMissing)
	}

	log.Info("checking primary keys on development database")
	devMissing, err := schema.CheckPrimaryKeys(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("dev primary key check: %w", err)
	}
	if len(devMissing) > 0 {
		return fmt.Errorf("dev tables missing primary keys: %v", devMissing)
	}

	// Print summary to console
	fmt.Println("Config loaded.")
	fmt.Printf("Prod: %s:%d/%s\n", cfg.Prod.Host, cfg.Prod.Port, cfg.Prod.Database)
	fmt.Printf("Dev : %s:%d/%s\n", cfg.Dev.Host, cfg.Dev.Port, cfg.Dev.Database)
	fmt.Printf("Output directory ready: %s\n", cfg.Output.Dir)
	fmt.Printf("Ignore tables: %v\n", cfg.Ignore.Tables)
	fmt.Printf("Ignore columns: %v\n", cfg.Ignore.Columns)
	fmt.Println("Connections OK. Primary keys verified.")

	log.Info("configuration check complete")
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
	generateHTML := fs.Bool("html", false, "Generate interactive HTML report")
	batchSizeFlag := fs.Int("batch-size", 0, "Rows per keyset-paginated query (0 = use config default)")
	parallelFlag := fs.Int("parallel", 0, "Max tables hashed concurrently (0 = use config default)")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting full diff (schema + data)")

	// Initialize progress manager (disabled in verbose mode to avoid conflicts)
	progressMgr := progress.NewManager(progress.Config{
		Enabled:     !*verbose,
		ShowMetrics: true,
	})
	defer func() {
		progressMgr.Finish()
		// Print metrics summary if available
		if metrics := progressMgr.GetMetrics(); metrics != nil {
			summary := metrics.Summary()
			if summary != "" {
				fmt.Println(summary)
			}
		}
	}()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	ctx := logger.ToContext(context.Background(), log)
	ctx = progress.ToContext(ctx, progressMgr)

	log.Info("opening database connections")
	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Schema diff first
	log.Info("loading database schemas")
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	log.Info("comparing schemas")
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}
	// Capture drift but continue — data diff, summary, and HTML must still be written.
	// The error is returned at the end so the exit code is still non-zero.
	var schemaDriftErr error
	if schemaDiff.HasDrift() {
		log.Warn("schema drift detected")
		fmt.Fprintf(os.Stderr, "Warning: schema drift detected; see %s and %s\n", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
		fmt.Fprintf(os.Stderr, "Warning: continuing with data diff. Only tables with matching schemas will be included.\n")
		schemaDriftErr = fmt.Errorf("schema drift detected; see %s and %s", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
	}

	// Data diff
	log.Info("starting data comparison")
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)

	batchSize := resolveBatchSize(*batchSizeFlag, cfg.Performance.HashBatchSize)
	maxParallel := resolveParallel(*parallelFlag, cfg.Performance.MaxParallelTables)
	log.Debug("hashing parameters", "batch_size", batchSize, "max_parallel", maxParallel)

	// Build the set of shared tables (exist in both prod and dev).
	sharedProdTables := make(map[string]schema.Table)
	sharedDevTables := make(map[string]schema.Table)
	for name, prodTable := range prodSchema.Tables {
		if devTable, ok := devSchema.Tables[name]; ok {
			sharedProdTables[name] = prodTable
			sharedDevTables[name] = devTable
		}
	}

	log.Info("hashing production tables", "count", len(sharedProdTables))
	prodHashes, err := hashTablesParallel(ctx, sharedProdTables, prodDB, cfg.Prod.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash prod tables: %w", err)
	}

	log.Info("hashing development tables", "count", len(sharedDevTables))
	devHashes, err := hashTablesParallel(ctx, sharedDevTables, devDB, cfg.Dev.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash dev tables: %w", err)
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	tablesScanned := len(sharedProdTables)

	if err := content.WriteReportsWithInfo(dataDiff, conflicts, cfg.Output.Dir, "OK", tablesScanned, ""); err != nil {
		return fmt.Errorf("write content diff: %w", err)
	}

	log.Info("full diff complete", "tables_scanned", tablesScanned, "has_changes", dataDiff.HasChanges())
	fmt.Println("Schema OK. Data diff complete.")
	if dataDiff.HasChanges() {
		fmt.Printf("Changes detected. See %s, %s, and %s\n", filepath.Join(cfg.Output.Dir, "content_diff.json"), filepath.Join(cfg.Output.Dir, "conflicts.json"), filepath.Join(cfg.Output.Dir, "summary.txt"))
		if conflicts.HasConflicts() {
			fmt.Printf("Warning: %d conflicts detected. Review %s\n", len(conflicts.Conflicts), filepath.Join(cfg.Output.Dir, "conflicts.json"))
		}
	} else {
		fmt.Println("No data differences found.")
	}

	// Generate HTML report if requested
	if *generateHTML {
		htmlPath := filepath.Join(cfg.Output.Dir, "report.html")
		reportData := htmlreport.BuildReportData(
			fmt.Sprintf("%s:%d/%s", cfg.Prod.Host, cfg.Prod.Port, cfg.Prod.Database),
			fmt.Sprintf("%s:%d/%s", cfg.Dev.Host, cfg.Dev.Port, cfg.Dev.Database),
			&schemaDiff,
			&dataDiff,
			&conflicts,
			nil, // No resolution info for diff command
			nil, // No resolutions for diff command
			"",  // No migration SQL for diff command
			"",  // No migration pack for diff command
			tablesScanned,
			nil,
		)

		generator := htmlreport.NewGenerator(nil)
		if err := generator.GenerateReport(reportData, htmlPath); err != nil {
			return fmt.Errorf("generate HTML report: %w", err)
		}
		fmt.Printf("HTML report generated: %s\n", htmlPath)
	}

	// Return deferred schema drift error so exit code is non-zero when drift was found,
	// even though all output files have been written successfully.
	return schemaDriftErr
}

// runGenPack performs the "gen-pack" command: it loads configuration, compares prod and dev schemas, and generates a migration pack for any detected data differences while writing schema and data reports to the configured output directory.
// It opens prod and dev database connections, ensures the output directory exists, fails if schema drift is detected (after writing schema reports), computes per-table content hashes with configured ignore rules, builds the data diff and any conflicts, generates a migration pack when changes exist, writes data reports including the pack path and conflicts, and prints a brief status summary.
// It returns a non-nil error for failures during configuration loading, database connections, schema loading or reporting, hashing, pack generation, or report writing; it also returns an error when schema drift is detected so the pack is not produced.
func runGenPack(args []string) error {
	fs := flag.NewFlagSet("gen-pack", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	generateHTML := fs.Bool("html", false, "Generate interactive HTML report")
	resumeCheckpoint := fs.Bool("resume", false, "Resume from checkpoint if available")
	batchSizeFlag := fs.Int("batch-size", 0, "Rows per keyset-paginated query (0 = use config default)")
	parallelFlag := fs.Int("parallel", 0, "Max tables hashed concurrently (0 = use config default)")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting migration pack generation")

	// Initialize progress manager (disabled in verbose mode to avoid conflicts)
	progressMgr := progress.NewManager(progress.Config{
		Enabled:     !*verbose,
		ShowMetrics: true,
	})
	defer func() {
		progressMgr.Finish()
		// Print metrics summary if available
		if metrics := progressMgr.GetMetrics(); metrics != nil {
			summary := metrics.Summary()
			if summary != "" {
				fmt.Println(summary)
			}
		}
	}()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	// Initialize checkpoint manager
	checkpointMgr := checkpoint.NewManager(cfg.Output.Dir)
	ctx := logger.ToContext(context.Background(), log)
	ctx = progress.ToContext(ctx, progressMgr)
	ctx = checkpoint.ToContext(ctx, checkpointMgr)

	// Handle resume from checkpoint
	var checkpointState *checkpoint.State
	if *resumeCheckpoint && checkpointMgr.HasCheckpoint() {
		log.Info("checkpoint found, attempting to resume")
		state, err := checkpointMgr.Load()
		if err != nil {
			return fmt.Errorf("load checkpoint: %w", err)
		}
		if state != nil {
			// Validate checkpoint
			if err := checkpoint.Validate(state, cfg, checkpoint.ResumeOptions{
				ValidateConfig: true,
			}); err != nil {
				return fmt.Errorf("validate checkpoint: %w", err)
			}
			checkpointState = state
			log.Info("resuming from checkpoint", "operation", state.Operation, "created_at", state.CreatedAt)
		}
	} else if !*resumeCheckpoint {
		// Create new checkpoint state if not resuming
		state, err := checkpoint.NewState(checkpoint.OperationTypeGeneratePack, cfg.Output.Dir, cfg)
		if err != nil {
			return fmt.Errorf("create checkpoint state: %w", err)
		}
		state.GeneratePackState = &checkpoint.GeneratePackState{
			CompletedTables: []string{},
			Statements:      []string{},
		}
		if err := checkpointMgr.Save(state); err != nil {
			log.Warn("failed to create checkpoint", logger.FieldError, err.Error())
		}
		checkpointState = state
	}

	// Cleanup checkpoint on success
	defer func() {
		if checkpointMgr != nil && checkpointState != nil {
			if err := checkpointMgr.Delete(); err != nil {
				log.Warn("failed to cleanup checkpoint", logger.FieldError, err.Error())
			} else {
				log.Debug("checkpoint cleaned up successfully")
			}
		}
	}()

	log.Info("opening database connections")
	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Schema diff first
	log.Info("loading database schemas")
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	log.Info("comparing schemas", "prod_tables", len(prodSchema.Tables), "dev_tables", len(devSchema.Tables))
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)
	if err := schema.WriteReports(schemaDiff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}
	if schemaDiff.HasDrift() {
		log.Warn("schema drift detected", "modified_tables", len(schemaDiff.Tables))
		fmt.Fprintf(os.Stderr, "Warning: schema drift detected; see %s and %s\n", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
		fmt.Fprintf(os.Stderr, "Warning: continuing with pack generation. Only tables with matching schemas will be included.\n")
	}

	// Data diff
	log.Info("starting table hashing")
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)

	batchSize := resolveBatchSize(*batchSizeFlag, cfg.Performance.HashBatchSize)
	maxParallel := resolveParallel(*parallelFlag, cfg.Performance.MaxParallelTables)
	log.Debug("hashing parameters", "batch_size", batchSize, "max_parallel", maxParallel)

	// Build the set of shared tables.
	sharedProdTables := make(map[string]schema.Table)
	sharedDevTables := make(map[string]schema.Table)
	for name, prodTable := range prodSchema.Tables {
		if devTable, ok := devSchema.Tables[name]; ok {
			sharedProdTables[name] = prodTable
			sharedDevTables[name] = devTable
		}
	}

	log.Info("hashing production tables", "count", len(sharedProdTables))
	prodHashes, err := hashTablesParallel(ctx, sharedProdTables, prodDB, cfg.Prod.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash prod tables: %w", err)
	}

	log.Info("hashing development tables", "count", len(sharedDevTables))
	devHashes, err := hashTablesParallel(ctx, sharedDevTables, devDB, cfg.Dev.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash dev tables: %w", err)
	}

	dataDiff, conflicts := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	tablesScanned := len(sharedProdTables)

	// Apply conflict resolution if conflicts exist
	var resolutions []resolve.Resolution
	var filteredDiff content.DataDiff
	var excludedCounts map[resolve.Decision]int

	if conflicts.HasConflicts() {
		// Resolve conflicts based on configured strategies
		resolutions = resolve.Conflicts(conflicts, cfg)

		// Filter the data diff based on resolutions
		filteredDiff, excludedCounts = resolve.FilterDataDiffByResolutions(dataDiff, resolutions)

		// Print resolution summary
		summary := resolve.BuildResolutionSummary(resolutions)
		fmt.Printf("\nConflict Resolution Summary:\n")
		fmt.Printf("  Total conflicts: %d\n", summary.TotalConflicts)
		if summary.ByDecision[resolve.DecisionUseDev] > 0 {
			fmt.Printf("  Auto-resolved (theirs -> use dev): %d\n", summary.ByDecision[resolve.DecisionUseDev])
		}
		if summary.ByDecision[resolve.DecisionKeepProd] > 0 {
			fmt.Printf("  Auto-resolved (ours -> keep prod): %d\n", summary.ByDecision[resolve.DecisionKeepProd])
		}
		if summary.UnresolvedCount > 0 {
			fmt.Printf("  Pending manual review: %d\n", summary.UnresolvedCount)
		}
		fmt.Println()
	} else {
		filteredDiff = dataDiff
		excludedCounts = make(map[resolve.Decision]int)
	}

	packPath, err := content.GeneratePack(ctx, cfg.Prod.Driver, devDB, cfg.Dev.Database, prodSchema, devSchema, schemaDiff, filteredDiff, ignoreColumn, cfg.Output.Dir)
	if err != nil {
		return fmt.Errorf("generate pack: %w", err)
	}

	// Build resolution info for reports if resolutions exist
	var resInfo *content.ResolutionInfo
	if len(resolutions) > 0 {
		summary := resolve.BuildResolutionSummary(resolutions)
		byDecision := make(map[string]int)
		for d, count := range summary.ByDecision {
			byDecision[string(d)] = count
		}
		resInfo = content.BuildResolutionInfo(
			summary.TotalConflicts,
			summary.ResolvedCount,
			summary.UnresolvedCount,
			byDecision,
			summary.ByTable,
		)
	}

	if err := content.WriteReportsWithResolutions(dataDiff, conflicts, cfg.Output.Dir, "OK", tablesScanned, packPath, resInfo); err != nil {
		return fmt.Errorf("write content diff: %w", err)
	}

	if schemaDiff.HasDrift() {
		fmt.Println("Schema drift detected. Data diff complete.")
	} else {
		fmt.Println("Schema OK. Data diff complete.")
	}
	if filteredDiff.HasChanges() {
		fmt.Printf("Changes detected. Pack written to %s\n", packPath)
		if excludedCounts[resolve.DecisionKeepProd] > 0 {
			fmt.Printf("  %d conflicts excluded (ours strategy - keeping prod values)\n", excludedCounts[resolve.DecisionKeepProd])
		}
		if excludedCounts[resolve.DecisionPending] > 0 {
			fmt.Printf("  Warning: %d conflicts excluded (manual review required)\n", excludedCounts[resolve.DecisionPending])
		}
	} else if dataDiff.HasChanges() {
		// Original diff had changes but filtered diff doesn't
		fmt.Println("All data changes were excluded by conflict resolution.")
		fmt.Printf("  %d conflicts kept in prod (ours strategy)\n", excludedCounts[resolve.DecisionKeepProd])
		if excludedCounts[resolve.DecisionPending] > 0 {
			fmt.Printf("  %d conflicts pending manual review\n", excludedCounts[resolve.DecisionPending])
		}
	} else {
		fmt.Println("No data differences found. Pack not required.")
	}

	// Generate HTML report if requested
	if *generateHTML {
		htmlPath := filepath.Join(cfg.Output.Dir, "report.html")

		// Read migration pack SQL if it was generated
		var migrationSQL string
		if packPath != "" {
			if sqlBytes, err := os.ReadFile(packPath); err == nil {
				migrationSQL = string(sqlBytes)
			}
		}

		reportData := htmlreport.BuildReportData(
			fmt.Sprintf("%s:%d/%s", cfg.Prod.Host, cfg.Prod.Port, cfg.Prod.Database),
			fmt.Sprintf("%s:%d/%s", cfg.Dev.Host, cfg.Dev.Port, cfg.Dev.Database),
			&schemaDiff,
			&dataDiff,
			&conflicts,
			resInfo,
			resolutions,
			migrationSQL,
			filepath.Base(packPath),
			tablesScanned,
			nil,
		)

		generator := htmlreport.NewGenerator(nil)
		if err := generator.GenerateReport(reportData, htmlPath); err != nil {
			return fmt.Errorf("generate HTML report: %w", err)
		}
		fmt.Printf("HTML report generated: %s\n", htmlPath)
	}

	return nil
}

// runApply parses command-line flags and applies or validates a migration pack against
// the configured production database.
//
// It accepts the following flags on args:
//
//	--pack   Path to the migration pack SQL file (required).
//	--dry-run  Validate the SQL without executing it.
//	--config Path to the configuration file (default "deepdiffdb.config.yaml").
//
// runApply returns an error when flag parsing fails, the required --pack is missing,
// the configuration cannot be loaded, the target database connection cannot be opened,
// or when validation/application of the pack fails.
func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	packPath := fs.String("pack", "", "Path to migration pack SQL file (required)")
	dryRun := fs.Bool("dry-run", false, "Validate SQL without executing")
	resumeCheckpoint := fs.Bool("resume", false, "Resume from checkpoint if available")
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *packPath == "" {
		return fmt.Errorf("--pack flag is required")
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting migration pack application")

	// Handle resume from checkpoint if requested
	if *resumeCheckpoint {
		log.Info("resume flag enabled - will attempt to resume from checkpoint if available")
	}

	// Initialize progress manager (disabled in verbose mode to avoid conflicts)
	progressMgr := progress.NewManager(progress.Config{
		Enabled:     !*verbose,
		ShowMetrics: true,
	})
	defer func() {
		progressMgr.Finish()
		// Print metrics summary if available
		if metrics := progressMgr.GetMetrics(); metrics != nil {
			summary := metrics.Summary()
			if summary != "" {
				fmt.Println(summary)
			}
		}
	}()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	// Initialize checkpoint manager for apply command
	checkpointMgr := checkpoint.NewManager(cfg.Output.Dir)
	ctx := logger.ToContext(context.Background(), log)
	ctx = progress.ToContext(ctx, progressMgr)
	ctx = checkpoint.ToContext(ctx, checkpointMgr)

	// Handle resume from checkpoint if requested
	if *resumeCheckpoint && checkpointMgr.HasCheckpoint() {
		log.Info("checkpoint found, will attempt to resume")
	} else if *resumeCheckpoint {
		log.Info("resume flag enabled but no checkpoint found - starting fresh")
	}

	// Cleanup checkpoint on success
	defer func() {
		if checkpointMgr != nil {
			if err := checkpointMgr.Delete(); err != nil {
				log.Warn("failed to cleanup checkpoint", logger.FieldError, err.Error())
			}
		}
	}()

	// Apply to prod database
	log.Info("connecting to target database", logger.FieldDatabase, cfg.Prod.Database)
	targetDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "target")
	if err != nil {
		return fmt.Errorf("target connection failed: %w", err)
	}
	defer targetDB.Close()

	if *dryRun {
		log.Info("dry-run mode: validating migration pack", logger.FieldPath, *packPath)
		fmt.Println("Dry-run mode: validating SQL...")
		if err := content.ApplyPack(ctx, targetDB, *packPath, true); err != nil {
			return fmt.Errorf("dry-run validation failed: %w", err)
		}
		log.Info("dry-run validation passed")
		fmt.Println("Dry-run validation passed. SQL is valid.")
		return nil
	}

	log.Info("applying migration pack", logger.FieldPath, *packPath)
	fmt.Printf("Applying migration pack: %s\n", *packPath)
	if err := content.ApplyPack(ctx, targetDB, *packPath, false); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}
	log.Info("migration pack applied successfully")
	fmt.Println("Migration pack applied successfully.")
	return nil
}

// runResolveConflicts implements the interactive conflict resolution command.
// It loads conflicts from conflicts.json, optionally resumes from a saved resolutions file,
// and allows users to interactively resolve conflicts one by one with full row data comparison.
func runResolveConflicts(args []string) error {
	fs := flag.NewFlagSet("resolve-conflicts", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	conflictsPath := fs.String("conflicts", "", "Path to conflicts.json file (default: <output-dir>/conflicts.json)")
	resolutionsPath := fs.String("resolutions", "", "Path to resolutions.json for persistence (default: <output-dir>/resolutions.json)")
	autoMode := fs.Bool("auto", false, "Apply configured strategies without prompts")
	resumeMode := fs.Bool("resume", false, "Resume from existing resolutions file")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting conflict resolution")

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	// Set default paths
	if *conflictsPath == "" {
		*conflictsPath = filepath.Join(cfg.Output.Dir, "conflicts.json")
	}
	if *resolutionsPath == "" {
		*resolutionsPath = filepath.Join(cfg.Output.Dir, "resolutions.json")
	}

	// Load conflicts
	log.Info("loading conflicts file", logger.FieldPath, *conflictsPath)
	conflictsData, err := os.ReadFile(*conflictsPath)
	if err != nil {
		return fmt.Errorf("read conflicts file: %w", err)
	}

	var conflicts content.Conflicts
	if err := json.Unmarshal(conflictsData, &conflicts); err != nil {
		return fmt.Errorf("parse conflicts file: %w", err)
	}

	if !conflicts.HasConflicts() {
		log.Info("no conflicts to resolve")
		fmt.Println("No conflicts to resolve.")
		return nil
	}

	log.Info("conflicts loaded", "total_conflicts", len(conflicts.Conflicts))

	// Initialize resolutions
	var resolutions []resolve.Resolution

	if *resumeMode {
		// Load existing resolutions and merge with current conflicts
		saved, err := resolve.LoadResolutions(*resolutionsPath)
		if err != nil {
			fmt.Printf("Warning: could not load resolutions file: %v\n", err)
			fmt.Println("Starting fresh resolution session.")
			resolutions = resolve.Conflicts(conflicts, nil) // All pending
		} else {
			resolutions = resolve.MergeResolutions(saved.Resolutions, conflicts)
			fmt.Printf("Resumed from saved resolutions. %d conflicts loaded.\n", len(resolutions))
		}
	} else {
		// Start fresh - create pending resolutions for all conflicts
		resolutions = make([]resolve.Resolution, 0, len(conflicts.Conflicts))
		for _, c := range conflicts.Conflicts {
			resolutions = append(resolutions, resolve.Resolution{
				Conflict: c,
				Strategy: resolve.StrategyManual,
				Decision: resolve.DecisionPending,
				Resolved: false,
			})
		}
	}

	// Auto mode: apply configured strategies without prompts
	if *autoMode {
		log.Info("running in auto mode - applying configured strategies")
		resolutions = resolve.Conflicts(conflicts, cfg)
		if err := resolve.SaveResolutions(resolutions, *resolutionsPath); err != nil {
			return fmt.Errorf("save resolutions: %w", err)
		}

		summary := resolve.BuildResolutionSummary(resolutions)
		display := cli.NewDisplay()
		display.PrintSummary(summary, *resolutionsPath)
		log.Info("auto resolution complete", "resolved", summary.ResolvedCount, "unresolved", summary.UnresolvedCount)
		return nil
	}

	// Interactive mode: connect to databases for row data
	log.Info("running in interactive mode")
	ctx := logger.ToContext(context.Background(), log)
	// Note: No progress manager in interactive mode (conflicts with prompts)

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

	// Load schemas for row fetching
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	display := cli.NewDisplay()
	prompter := cli.NewPrompter()

	// Show welcome message
	pendingCount := resolve.GetPendingCount(resolutions)
	display.PrintWelcome(len(resolutions), pendingCount)

	if pendingCount == 0 {
		display.PrintAllResolved()
		summary := resolve.BuildResolutionSummary(resolutions)
		display.PrintSummary(summary, *resolutionsPath)
		return nil
	}

	// Interactive resolution loop
	totalConflicts := len(resolutions)
	currentIndex := 0

	// Find first pending conflict
	for currentIndex < totalConflicts && resolutions[currentIndex].Resolved {
		currentIndex++
	}

	for currentIndex < totalConflicts {
		res := &resolutions[currentIndex]

		// Skip already resolved
		if res.Resolved {
			currentIndex++
			continue
		}

		// Calculate display position (counting only pending)
		displayPos := 0
		for i := 0; i <= currentIndex; i++ {
			if !resolutions[i].Resolved || i == currentIndex {
				displayPos++
			}
		}

		// Fetch row data for comparison
		prod, dev, err := resolve.FetchConflictRows(
			ctx, prodDB, devDB, cfg.Prod.Driver,
			prodSchema, devSchema, res.Conflict,
		)
		if err != nil {
			display.PrintError(fmt.Sprintf("fetch row data: %v", err))
			display.PrintInfo("Skipping to next conflict...")
			currentIndex++
			continue
		}

		// Display conflict
		display.PrintProgress(displayPos, pendingCount, res.Conflict.Table, res.Conflict.Key)

		// Compare rows and display
		diffs := resolve.CompareRows(prod, dev)
		display.PrintConflictComparison(prod, dev, diffs)

		// Prompt for resolution
		choice, err := prompter.PromptResolution(res.Conflict.Table)
		if err != nil {
			display.PrintError(fmt.Sprintf("prompt error: %v", err))
			continue
		}

		switch choice {
		case cli.ChoiceKeepProd:
			resolutions = resolve.UpdateResolution(resolutions, currentIndex, resolve.StrategyOurs, resolve.DecisionKeepProd)
			currentIndex++

		case cli.ChoiceUseDev:
			resolutions = resolve.UpdateResolution(resolutions, currentIndex, resolve.StrategyTheirs, resolve.DecisionUseDev)
			currentIndex++

		case cli.ChoiceSkip:
			currentIndex++

		case cli.ChoiceOursForTable:
			resolutions = resolve.ApplyBulkResolution(resolutions, res.Conflict.Table, false, resolve.StrategyOurs)
			count := countTableResolutions(resolutions, res.Conflict.Table, resolve.DecisionKeepProd)
			display.PrintBulkApplied("ours", count, res.Conflict.Table)
			// Skip to next table
			for currentIndex < totalConflicts && resolutions[currentIndex].Conflict.Table == res.Conflict.Table {
				currentIndex++
			}

		case cli.ChoiceTheirsForTable:
			resolutions = resolve.ApplyBulkResolution(resolutions, res.Conflict.Table, false, resolve.StrategyTheirs)
			count := countTableResolutions(resolutions, res.Conflict.Table, resolve.DecisionUseDev)
			display.PrintBulkApplied("theirs", count, res.Conflict.Table)
			// Skip to next table
			for currentIndex < totalConflicts && resolutions[currentIndex].Conflict.Table == res.Conflict.Table {
				currentIndex++
			}

		case cli.ChoiceOursForAll:
			resolutions = resolve.ApplyBulkResolution(resolutions, "", true, resolve.StrategyOurs)
			count := countDecisionResolutions(resolutions, resolve.DecisionKeepProd)
			display.PrintBulkApplied("ours", count, "all remaining conflicts")
			currentIndex = totalConflicts // Exit loop

		case cli.ChoiceTheirsForAll:
			resolutions = resolve.ApplyBulkResolution(resolutions, "", true, resolve.StrategyTheirs)
			count := countDecisionResolutions(resolutions, resolve.DecisionUseDev)
			display.PrintBulkApplied("theirs", count, "all remaining conflicts")
			currentIndex = totalConflicts // Exit loop

		case cli.ChoiceQuit:
			display.PrintSaving(*resolutionsPath)
			if err := resolve.SaveResolutions(resolutions, *resolutionsPath); err != nil {
				return fmt.Errorf("save resolutions: %w", err)
			}
			display.PrintSaved(*resolutionsPath)
			pendingCount = resolve.GetPendingCount(resolutions)
			display.PrintSkipped(pendingCount)
			return nil

		case cli.ChoiceInvalid:
			display.PrintError("Invalid choice. Please try again.")
			continue
		}

		// Save after each decision
		if err := resolve.SaveResolutions(resolutions, *resolutionsPath); err != nil {
			display.PrintWarning(fmt.Sprintf("could not save resolutions: %v", err))
		}
	}

	// Final summary
	summary := resolve.BuildResolutionSummary(resolutions)
	display.PrintSummary(summary, *resolutionsPath)
	log.Info("conflict resolution complete", "total", summary.TotalConflicts, "resolved", summary.ResolvedCount, "unresolved", summary.UnresolvedCount)

	return nil
}

// countTableResolutions counts resolutions for a specific table and decision.
func countTableResolutions(resolutions []resolve.Resolution, table string, decision resolve.Decision) int {
	count := 0
	for _, res := range resolutions {
		if res.Conflict.Table == table && res.Decision == decision {
			count++
		}
	}
	return count
}

// countDecisionResolutions counts resolutions with a specific decision.
func countDecisionResolutions(resolutions []resolve.Resolution, decision resolve.Decision) int {
	count := 0
	for _, res := range resolutions {
		if res.Decision == decision {
			count++
		}
	}
	return count
}

// runSchemaDiff parses flags for the "schema-diff" command, loads configuration, opens production and development databases, loads their schemas, writes schema diff reports to the configured output directory, and returns an error if schema drift is detected or any operation fails.
func runSchemaDiff(args []string) error {
	fs := flag.NewFlagSet("schema-diff", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting schema diff")

	// Initialize progress manager (disabled in verbose mode to avoid conflicts)
	progressMgr := progress.NewManager(progress.Config{
		Enabled:     !*verbose,
		ShowMetrics: true,
	})
	defer func() {
		progressMgr.Finish()
		// Print metrics summary if available
		if metrics := progressMgr.GetMetrics(); metrics != nil {
			summary := metrics.Summary()
			if summary != "" {
				fmt.Println(summary)
			}
		}
	}()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	ctx := logger.ToContext(context.Background(), log)
	ctx = progress.ToContext(ctx, progressMgr)

	log.Info("opening database connections")
	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	log.Info("loading database schemas")
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	log.Info("comparing schemas", "prod_tables", len(prodSchema.Tables), "dev_tables", len(devSchema.Tables))
	diff := schema.DiffSchemas(prodSchema, devSchema)

	if err := schema.WriteReports(diff, cfg.Output.Dir); err != nil {
		return fmt.Errorf("write schema diff: %w", err)
	}

	if diff.HasDrift() {
		log.Warn("schema drift detected")
		return fmt.Errorf("schema drift detected; see %s and %s", filepath.Join(cfg.Output.Dir, "schema_diff.json"), filepath.Join(cfg.Output.Dir, "schema_diff.txt"))
	}

	log.Info("schema diff complete - no drift detected")
	fmt.Println("Schema match confirmed. No drift detected.")
	return nil
}

// runSchemaMigrate generates a standalone schema migration script based on schema differences
// between production and development databases.
//
// It parses flags (--config, --dry-run), loads configuration, opens both database connections,
// loads schemas, computes schema diff, generates migration SQL, and writes to output directory.
// runSchemaMigrate generates a standalone schema migration SQL file by comparing the production and development schemas.
// It accepts CLI flags (parsed from args) including --config and --dry-run; in dry-run mode the generated migration is printed instead of written to disk. It constructs migration options from the loaded config, connects to both databases, computes the schema diff, generates the migration using those options, writes the migration to output/schema_migration.sql when changes exist, and returns an error if any step fails.
func runSchemaMigrate(args []string) error {
	fs := flag.NewFlagSet("schema-migrate", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	dryRun := fs.Bool("dry-run", false, "Generate and validate migration without writing file")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Initialize logger
	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	log.Info("starting schema migration generation")

	// Initialize progress manager (disabled in verbose mode to avoid conflicts)
	progressMgr := progress.NewManager(progress.Config{
		Enabled:     !*verbose,
		ShowMetrics: true,
	})
	defer func() {
		progressMgr.Finish()
		// Print metrics summary if available
		if metrics := progressMgr.GetMetrics(); metrics != nil {
			summary := metrics.Summary()
			if summary != "" {
				fmt.Println(summary)
			}
		}
	}()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Debug("configuration loaded", "config_path", *configPath)

	ctx := logger.ToContext(context.Background(), log)
	ctx = progress.ToContext(ctx, progressMgr)

	log.Info("opening database connections")
	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection failed: %w", err)
	}
	defer prodDB.Close()

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection failed: %w", err)
	}
	defer devDB.Close()

	if err := os.MkdirAll(cfg.Output.Dir, 0o750); err != nil {
		return fmt.Errorf("ensure output dir: %w", err)
	}

	// Load schemas
	log.Info("loading database schemas")
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	// Compute schema diff
	log.Info("comparing schemas and generating migration SQL")
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)

	// Prepare migration options from config
	opts := &schema.MigrationOptions{
		AllowDropColumn:       cfg.Migration.AllowDropColumn,
		AllowDropTable:        cfg.Migration.AllowDropTable,
		AllowDropIndex:        cfg.Migration.AllowDropIndex,
		AllowDropForeignKey:   cfg.Migration.AllowDropForeignKey,
		AllowModifyPrimaryKey: cfg.Migration.AllowModifyPrimaryKey,
		ConfirmDestructive:    cfg.Migration.ConfirmDestructive,
	}

	// Generate migration script with proper dependency ordering
	migrationSQL, err := schema.GenerateMigrationWithSchemas(schemaDiff, cfg.Prod.Driver, opts, prodSchema)
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
		log.Info("no schema changes detected")
		fmt.Println("No schema changes detected. No migration file generated.")
		return nil
	}

	// Write migration file
	migrationPath := filepath.Join(cfg.Output.Dir, "schema_migration.sql")
	if err := os.WriteFile(migrationPath, []byte(migrationSQL), 0o600); err != nil {
		return fmt.Errorf("write migration file: %w", err)
	}

	log.Info("schema migration file generated", logger.FieldPath, migrationPath)
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
  check             Validate configuration and show quick summary
  schema-diff       Detect schema drift
  schema-migrate    Generate schema migration script
  diff              Full diff: schema + data (supports --html for interactive report)
  gen-pack          Generate SQL migration pack (supports --html, --resume)
  apply             Apply migration pack (supports --resume)
  resolve-conflicts Interactively resolve pending conflicts (supports --resume)
  version           Git-like versioning for DB diffs (init/commit/log/diff/rollback)

Global Flags:
  -v, --version   Show version information
  -h, --help      Show this help message

Common Flags (available on most commands):
  --verbose       Enable verbose logging (DEBUG level)
  --log-file      Write logs to file
  --log-level     Log level: debug, info, warn, error
  --log-format    Log format: text or json
  --resume        Resume from checkpoint if available (gen-pack, apply)
  --batch-size    Rows per paginated query, 0 = config default (diff, gen-pack)
  --parallel      Max tables hashed concurrently, 0 = config default (diff, gen-pack)

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

// ---------------------------------------------------------------------------
// version subcommand group
// ---------------------------------------------------------------------------

func runVersion(args []string) error {
	if len(args) == 0 {
		printVersionUsage()
		return nil
	}
	switch args[0] {
	case "init":
		return runVersionInit(args[1:])
	case "commit":
		return runVersionCommit(args[1:])
	case "log":
		return runVersionLog(args[1:])
	case "diff":
		return runVersionDiff(args[1:])
	case "rollback":
		return runVersionRollback(args[1:])
	case "branch":
		return runVersionBranch(args[1:])
	case "checkout":
		return runVersionCheckout(args[1:])
	case "tree":
		return runVersionTree(args[1:])
	case "-h", "--help", "help":
		printVersionUsage()
		return nil
	default:
		printVersionUsage()
		return fmt.Errorf("unknown version subcommand %q", args[0])
	}
}

func printVersionUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`DeepDiff DB — version subcommands

Usage:
  %[1]s version <subcommand> [flags]

Subcommands:
  init                    Initialise a version repository in .deepdiffdb/
  commit                  Run a diff and store the result as a new commit
  log                     Show commit history
  diff  <hash1> <hash2>   Compare schema evolution between two commits
  rollback <hash>         Generate rollback SQL for a commit
  branch [<name>]         List branches, or create a new one
  checkout <branch>       Switch to a branch
  tree                    Show ASCII commit graph for all branches

Flags for init:
  --skip-auth  Skip GitHub authentication prompt

Flags for commit:
  --config   Path to config file (default: deepdiffdb.config.yaml)
  --message  Commit message (required)
  --author   Author name (used only when not authenticated via GitHub)

Flags for log:
  --limit    Maximum number of commits to show (default: 20)

Flags for rollback:
  --driver   Database driver override (mysql/postgres/sqlite/mssql/oracle)
  --out      Write SQL to file instead of stdout

Flags for branch:
  --from     Hash to branch from (default: current HEAD)
`, exe)
}

// runVersionInit initialises a .deepdiffdb/ repo in the current directory.
func runVersionInit(args []string) error {
	fs := flag.NewFlagSet("version init", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Directory to initialise (default: current directory)")
	skipAuth := fs.Bool("skip-auth", false, "Skip GitHub authentication (use --author flag on commits instead)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if vcs.IsInitialized(*dir) {
		fmt.Println("Version repository already initialised.")
		return nil
	}
	if err := vcs.Init(*dir); err != nil {
		return err
	}
	fmt.Printf("Initialised version repository in %s/%s\n", *dir, vcs.RepoDirName)

	if *skipAuth {
		fmt.Println("Skipping GitHub authentication. Use --author on version commit to set your name.")
		return nil
	}

	clientID := vcs.ResolveClientID()
	if clientID == "" {
		fmt.Println("\nNote: GitHub authentication unavailable (DEEPDIFFDB_GITHUB_CLIENT_ID not set).")
		fmt.Println("      Use --author on version commit to identify yourself.")
		return nil
	}

	fmt.Print("\nAuthenticate with GitHub to verify commit authorship? [Y/n]: ")
	var answer string
	fmt.Scanln(&answer) //nolint:errcheck
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" {
		fmt.Println("Skipped. Use --author on version commit to set your name.")
		return nil
	}

	username, err := vcs.RunGitHubDeviceFlow(clientID)
	if err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}
	if err := vcs.SaveIdentity(*dir, username); err != nil {
		return fmt.Errorf("saving identity: %w", err)
	}
	fmt.Printf("Authenticated as github:%s — your commits will be signed automatically.\n", username)
	return nil
}

// runVersionCommit runs a full schema+data diff and stores it as a new commit.
func runVersionCommit(args []string) error {
	fs := flag.NewFlagSet("version commit", flag.ContinueOnError)
	configPath := fs.String("config", "deepdiffdb.config.yaml", "Path to configuration file")
	message := fs.String("message", "", "Commit message (required)")
	author := fs.String("author", "", "Author name")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	logFile := fs.String("log-file", "", "Write logs to file")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := fs.String("log-format", "text", "Log format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *message == "" {
		return fmt.Errorf("--message is required")
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	log, logCloser, err := initializeLogger(*verbose, *logFile, *logLevel, *logFormat)
	if err != nil {
		return err
	}
	if logCloser != nil {
		defer logCloser.Close()
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := logger.ToContext(context.Background(), log)

	prodDB, err := openDatabaseWithSpinner(ctx, cfg.Prod, "production")
	if err != nil {
		return fmt.Errorf("prod connection: %w", err)
	}
	defer prodDB.Close()

	devDB, err := openDatabaseWithSpinner(ctx, cfg.Dev, "development")
	if err != nil {
		return fmt.Errorf("dev connection: %w", err)
	}
	defer devDB.Close()

	log.Info("loading schemas")
	prodSchema, err := schema.LoadSchema(ctx, prodDB, cfg.Prod.Driver, cfg.Prod.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load prod schema: %w", err)
	}
	devSchema, err := schema.LoadSchema(ctx, devDB, cfg.Dev.Driver, cfg.Dev.Database, cfg.Ignore.Tables)
	if err != nil {
		return fmt.Errorf("load dev schema: %w", err)
	}

	log.Info("comparing schemas")
	schemaDiff := schema.DiffSchemas(prodSchema, devSchema)

	log.Info("comparing data")
	ignoreColumn := content.IgnoreMatcher(cfg.Ignore.Columns)
	batchSize := resolveBatchSize(0, cfg.Performance.HashBatchSize)
	maxParallel := resolveParallel(0, cfg.Performance.MaxParallelTables)

	sharedProd := make(map[string]schema.Table)
	sharedDev := make(map[string]schema.Table)
	for name, pt := range prodSchema.Tables {
		if dt, ok := devSchema.Tables[name]; ok {
			sharedProd[name] = pt
			sharedDev[name] = dt
		}
	}

	prodHashes, err := hashTablesParallel(ctx, sharedProd, prodDB, cfg.Prod.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash prod tables: %w", err)
	}
	devHashes, err := hashTablesParallel(ctx, sharedDev, devDB, cfg.Dev.Driver, ignoreColumn, batchSize, maxParallel)
	if err != nil {
		return fmt.Errorf("hash dev tables: %w", err)
	}
	dataDiff, _ := content.BuildDataDiff(prodSchema, devSchema, prodHashes, devHashes)

	head, err := vcs.ReadHEAD(dir)
	if err != nil {
		return fmt.Errorf("read HEAD: %w", err)
	}

	// Resolve author: verified GitHub identity takes priority over the --author flag.
	// If neither is set, fall back to $USER for convenience. Unauthenticated commits
	// are allowed but the author is clearly unverified.
	authorName, err := vcs.LoadIdentity(dir)
	if err != nil {
		return fmt.Errorf("loading identity: %w", err)
	}
	if authorName != "" {
		authorName = "github:" + authorName
		if *author != "" {
			fmt.Fprintf(os.Stderr, "Warning: --author flag ignored; using verified identity %s\n", authorName)
		}
	} else {
		authorName = *author
		if authorName == "" {
			authorName = os.Getenv("USER")
		}
		if authorName == "" {
			return fmt.Errorf("no author set: authenticate with 'version init' or pass --author <name>")
		}
	}

	c := &vcs.Commit{
		Parent:     head,
		Timestamp:  nowUTC(),
		Author:     authorName,
		Message:    *message,
		Driver:     cfg.Prod.Driver,
		SchemaDiff: schemaDiff,
		DataDiff:   dataDiff,
		ProdSchema: prodSchema,
		DevSchema:  devSchema,
	}

	hash, err := vcs.ComputeHash(c)
	if err != nil {
		return fmt.Errorf("compute commit hash: %w", err)
	}

	if err := vcs.SaveCommit(dir, c); err != nil {
		return fmt.Errorf("save commit: %w", err)
	}

	fmt.Printf("[%s] %s\n", hash[:8], *message)
	if schemaDiff.HasDrift() {
		fmt.Println("  Schema drift detected.")
	}
	if dataDiff.HasChanges() {
		fmt.Println("  Data differences detected.")
	}
	if !schemaDiff.HasDrift() && !dataDiff.HasChanges() {
		fmt.Println("  No differences found (clean snapshot).")
	}
	return nil
}

// runVersionLog prints the commit history from HEAD backwards.
func runVersionLog(args []string) error {
	fs := flag.NewFlagSet("version log", flag.ContinueOnError)
	limit := fs.Int("limit", 20, "Maximum number of commits to display")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	head, err := vcs.ReadHEAD(dir)
	if err != nil {
		return fmt.Errorf("read HEAD: %w", err)
	}
	if head == "" {
		fmt.Println("No commits yet.")
		return nil
	}

	hash := head
	for i := 0; i < *limit && hash != ""; i++ {
		c, err := vcs.LoadCommit(dir, hash)
		if err != nil {
			return fmt.Errorf("load commit: %w", err)
		}
		schemaDriftMark := ""
		if c.SchemaDiff.HasDrift() {
			schemaDriftMark = " [schema drift]"
		}
		dataMark := ""
		if c.DataDiff.HasChanges() {
			dataMark = " [data changes]"
		}
		fmt.Printf("commit %s\n", c.Hash)
		fmt.Printf("Author: %s\n", c.Author)
		fmt.Printf("Date:   %s\n", c.Timestamp.Format("2006-01-02 15:04:05 -0700"))
		fmt.Printf("\n    %s%s%s\n\n", c.Message, schemaDriftMark, dataMark)
		hash = c.Parent
	}
	return nil
}

// runVersionDiff compares schema evolution between two commits.
func runVersionDiff(args []string) error {
	fs := flag.NewFlagSet("version diff", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: deepdiffdb version diff <hash1> <hash2>")
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	h1, h2 := fs.Arg(0), fs.Arg(1)
	c1, err := vcs.LoadCommit(dir, h1)
	if err != nil {
		return fmt.Errorf("load commit %s: %w", h1[:min8(h1)], err)
	}
	c2, err := vcs.LoadCommit(dir, h2)
	if err != nil {
		return fmt.Errorf("load commit %s: %w", h2[:min8(h2)], err)
	}

	diff, err := vcs.InterVersionDiff(c1, c2)
	if err != nil {
		return err
	}

	fmt.Printf("Schema evolution from %s (%s) → %s (%s)\n\n",
		c1.Hash[:8], c1.Message, c2.Hash[:8], c2.Message)

	if !diff.HasDrift() {
		fmt.Println("No schema changes between these commits.")
		return nil
	}

	for _, t := range diff.AddedTables {
		fmt.Printf("  + TABLE %s (added)\n", t.Name)
	}
	for _, name := range diff.RemovedTables {
		fmt.Printf("  - TABLE %s (removed)\n", name)
	}
	for _, td := range diff.Tables {
		if !td.HasDifferences || td.OnlyInDev || td.OnlyInProd {
			continue
		}
		fmt.Printf("  ~ TABLE %s\n", td.Name)
		for _, col := range td.AddedColumns {
			fmt.Printf("      + COLUMN %s %s\n", col.Name, col.DataType)
		}
		for _, col := range td.RemovedColumns {
			fmt.Printf("      - COLUMN %s %s\n", col.Name, col.DataType)
		}
		for _, cd := range td.ModifiedColumns {
			fmt.Printf("      ~ COLUMN %s  (%s → %s)\n", cd.Column, cd.ProdType, cd.DevType)
		}
		for _, idx := range td.AddedIndexes {
			fmt.Printf("      + INDEX %s\n", idx.Name)
		}
		for _, idx := range td.RemovedIndexes {
			fmt.Printf("      - INDEX %s\n", idx.Name)
		}
	}
	return nil
}

// runVersionRollback generates rollback SQL for a given commit.
func runVersionRollback(args []string) error {
	fs := flag.NewFlagSet("version rollback", flag.ContinueOnError)
	driverFlag := fs.String("driver", "", "Database driver override (mysql/postgres/sqlite/mssql/oracle)")
	outFlag := fs.String("out", "", "Write SQL to file (default: stdout)")
	if err := fs.Parse(reorderFlagsFirst(args, map[string]bool{"driver": true, "out": true})); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: deepdiffdb version rollback <hash>")
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	hash := fs.Arg(0)
	c, err := vcs.LoadCommit(dir, hash)
	if err != nil {
		return fmt.Errorf("load commit: %w", err)
	}

	sql, err := vcs.GenerateRollbackSQL(c, *driverFlag)
	if err != nil {
		return err
	}

	if *outFlag != "" {
		if err := os.WriteFile(*outFlag, []byte(sql), 0o600); err != nil {
			return fmt.Errorf("write rollback SQL: %w", err)
		}
		fmt.Printf("Rollback SQL written to %s\n", *outFlag)
		return nil
	}
	fmt.Println(sql)
	return nil
}

// runVersionBranch lists branches or creates a new one.
func runVersionBranch(args []string) error {
	fs := flag.NewFlagSet("version branch", flag.ContinueOnError)
	fromFlag := fs.String("from", "", "Hash to branch from (default: current HEAD)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	if fs.NArg() == 0 {
		// List branches.
		branches, err := vcs.ListBranches(dir)
		if err != nil {
			return err
		}
		if len(branches) == 0 {
			fmt.Println("(no branches)")
			return nil
		}
		for _, b := range branches {
			marker := "  "
			if b.Current {
				marker = "* "
			}
			tip := ""
			if b.Tip != "" {
				tip = " " + b.Tip[:min8(b.Tip)]
			}
			fmt.Printf("%s%s%s\n", marker, b.Name, tip)
		}
		return nil
	}

	// Create branch.
	name := fs.Arg(0)
	if err := vcs.CreateBranch(dir, name, *fromFlag); err != nil {
		return err
	}
	fmt.Printf("Branch %q created.\n", name)
	return nil
}

// runVersionCheckout switches to the named branch.
func runVersionCheckout(args []string) error {
	fs := flag.NewFlagSet("version checkout", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: deepdiffdb version checkout <branch>")
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	name := fs.Arg(0)
	if err := vcs.Checkout(dir, name); err != nil {
		return err
	}
	fmt.Printf("Switched to branch %q.\n", name)
	return nil
}

// runVersionTree renders an ASCII commit graph for all branches.
func runVersionTree(args []string) error {
	fs := flag.NewFlagSet("version tree", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir := "."
	if !vcs.IsInitialized(dir) {
		return fmt.Errorf("no version repository found; run 'deepdiffdb version init' first")
	}

	return vcs.RenderTree(dir, os.Stdout)
}

// ---------------------------------------------------------------------------
// helpers used by version commands
// ---------------------------------------------------------------------------

func nowUTC() time.Time {
	return time.Now().UTC()
}

func min8(s string) int {
	if len(s) >= 8 {
		return 8
	}
	return len(s)
}

// reorderFlagsFirst moves flag tokens (--flag [value]) before positional args
// so that the standard flag.FlagSet can parse them even when the user places
// positional args before flags (e.g. "rollback <hash> --out file").
//
// The knownValueFlags set lists flags that consume the next token as their
// value.  Bool flags (no value token) must NOT be listed there.
func reorderFlagsFirst(args []string, knownValueFlags map[string]bool) []string {
	var flagTokens []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 1 && arg[0] == '-' {
			// Strip leading dashes to get the flag name.
			name := arg[1:]
			if len(name) > 0 && name[0] == '-' {
				name = name[1:] // handle --flag
			}
			// Strip =value if present (--flag=value form).
			if idx := len(name); idx > 0 {
				for j, c := range name {
					if c == '=' {
						name = name[:j]
						break
					}
				}
			}
			flagTokens = append(flagTokens, arg)
			if knownValueFlags[name] && i+1 < len(args) && (len(args[i+1]) == 0 || args[i+1][0] != '-') {
				i++
				flagTokens = append(flagTokens, args[i])
			}
		} else {
			positionals = append(positionals, arg)
		}
	}
	return append(flagTokens, positionals...)
}
