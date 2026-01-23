// Package generate provides the CLI command for generating Terraform configuration
package generate

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/alanm/terraform-provider-typesense/internal/generator"
)

// Run executes the generate command with the given arguments
func Run(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)

	// Server connection flags
	host := fs.String("host", "", "Typesense server hostname")
	port := fs.Int("port", 8108, "Typesense server port")
	protocol := fs.String("protocol", "http", "Typesense server protocol (http or https)")
	apiKey := fs.String("api-key", "", "Typesense server API key")

	// Cloud connection flags
	cloudAPIKey := fs.String("cloud-api-key", "", "Typesense Cloud Management API key")

	// Output flags
	output := fs.String("output", "./generated", "Output directory for generated files")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: terraform-provider-typesense generate [options]

Generate Terraform configuration from an existing Typesense cluster.

Options:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  # Generate from local server
  terraform-provider-typesense generate \
    --host=localhost --port=8108 --protocol=http --api-key=xyz \
    --output=./generated

  # Generate from Typesense Cloud
  terraform-provider-typesense generate \
    --cloud-api-key=abc123 \
    --output=./generated

  # Generate from both server and cloud
  terraform-provider-typesense generate \
    --host=my-cluster.typesense.net --port=443 --protocol=https --api-key=xyz \
    --cloud-api-key=abc123 \
    --output=./generated
`)
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate that at least one connection is configured
	hasServerConfig := *host != "" && *apiKey != ""
	hasCloudConfig := *cloudAPIKey != ""

	if !hasServerConfig && !hasCloudConfig {
		return fmt.Errorf("at least one of server credentials (--host, --api-key) or cloud credentials (--cloud-api-key) is required")
	}

	// Set defaults for server config if host is provided
	if *host != "" && *apiKey == "" {
		return fmt.Errorf("--api-key is required when --host is specified")
	}

	// Create generator config
	cfg := &generator.Config{
		Host:        *host,
		Port:        *port,
		Protocol:    *protocol,
		APIKey:      *apiKey,
		CloudAPIKey: *cloudAPIKey,
		OutputDir:   *output,
	}

	// Run generator
	gen := generator.New(cfg)

	fmt.Printf("Generating Terraform configuration...\n")
	if hasServerConfig {
		fmt.Printf("  Server: %s://%s:%d\n", *protocol, *host, *port)
	}
	if hasCloudConfig {
		fmt.Printf("  Cloud: Typesense Cloud API\n")
	}
	fmt.Printf("  Output: %s\n", *output)
	fmt.Println()

	ctx := context.Background()
	if err := gen.Generate(ctx); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	fmt.Printf("Generated files:\n")
	fmt.Printf("  %s/main.tf     - Terraform configuration\n", *output)
	fmt.Printf("  %s/imports.sh  - Import commands script\n", *output)
	fmt.Println()
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. cd %s\n", *output)
	fmt.Printf("  2. Review and update main.tf (especially API key placeholder)\n")
	fmt.Printf("  3. terraform init\n")
	fmt.Printf("  4. ./imports.sh  # Import existing resources\n")
	fmt.Printf("  5. terraform plan  # Should show no changes\n")

	return nil
}
