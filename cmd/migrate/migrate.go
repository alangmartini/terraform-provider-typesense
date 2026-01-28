// Package migrate provides the CLI command for migrating Typesense data
package migrate

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alanm/terraform-provider-typesense/internal/migrator"
)

// Run executes the migrate command with the given arguments
func Run(args []string) error {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)

	// Source flags
	sourceDir := fs.String("source-dir", "", "Directory containing exported data from generate --include-data")

	// Target connection flags
	targetHost := fs.String("target-host", "", "Target Typesense server hostname")
	targetPort := fs.Int("target-port", 8108, "Target Typesense server port")
	targetProtocol := fs.String("target-protocol", "http", "Target Typesense server protocol (http or https)")
	targetAPIKey := fs.String("target-api-key", "", "Target Typesense server API key")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: terraform-provider-typesense migrate [options]

Import collections and documents to a target Typesense cluster from exported data.

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Migrate from exported data
  terraform-provider-typesense migrate \
    --source-dir=./migration \
    --target-host=target.typesense.net --target-port=443 --target-protocol=https \
    --target-api-key=$TARGET_API_KEY

Workflow:
  1. Export from source cluster:
     terraform-provider-typesense generate \
       --host=source.typesense.net --port=443 --protocol=https \
       --api-key=$SOURCE_KEY \
       --include-data \
       --output=./migration

  2. Import to target cluster:
     terraform-provider-typesense migrate \
       --source-dir=./migration \
       --target-host=target.typesense.net --target-port=443 --target-protocol=https \
       --target-api-key=$TARGET_KEY
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *sourceDir == "" {
		return fmt.Errorf("--source-dir is required")
	}
	if *targetHost == "" {
		return fmt.Errorf("--target-host is required")
	}
	if *targetAPIKey == "" {
		return fmt.Errorf("--target-api-key is required")
	}

	// Validate source directory exists
	if _, err := os.Stat(*sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", *sourceDir)
	}

	// Create migrator config
	cfg := &migrator.Config{
		SourceDir:      *sourceDir,
		TargetHost:     *targetHost,
		TargetPort:     *targetPort,
		TargetProtocol: *targetProtocol,
		TargetAPIKey:   *targetAPIKey,
	}

	// Run migration
	m := migrator.New(cfg)

	fmt.Printf("Migrating to target cluster...\n")
	fmt.Printf("  Source: %s\n", *sourceDir)
	fmt.Printf("  Target: %s://%s:%d\n", *targetProtocol, *targetHost, *targetPort)
	fmt.Println()

	ctx := context.Background()
	if err := m.Migrate(ctx); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("Migration complete!\n")

	return nil
}
