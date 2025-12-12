package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	case "schema-diff", "diff", "gen-pack", "apply":
		printNotImplemented(args[0])
		return nil
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

	fmt.Printf("Config loaded: prod(%s:%d/%s) dev(%s:%d/%s)\n",
		cfg.Prod.Driver, cfg.Prod.Port, cfg.Prod.Database,
		cfg.Dev.Driver, cfg.Dev.Port, cfg.Dev.Database,
	)
	fmt.Printf("Output directory: %s\n", cfg.Output.Dir)
	fmt.Printf("Ignore tables: %v\n", cfg.Ignore.Tables)
	fmt.Printf("Ignore columns: %v\n", cfg.Ignore.Columns)
	fmt.Println("Check command completed. Diff logic to be implemented.")
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
  schema-diff   Detect schema drift (coming soon)
  diff          Full diff: schema + data (coming soon)
  gen-pack      Generate SQL migration pack (coming soon)
  apply         Apply migration pack (coming soon)

Use "%[1]s <command> -h" for flags specific to that command.
`, exe)
}
