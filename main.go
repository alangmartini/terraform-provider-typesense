// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alanm/terraform-provider-typesense/cmd/generate"
	"github.com/alanm/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run "go generate" to format example terraform files and generate the docs for the registry/website

// If you do not have terraform installed, you can remove the formatting command, but its suggested to
// ensure the googledocumentation is formatted properly.
//go:generate terraform fmt -recursive ./examples/

// Run the docs generation tool, check its repository for more information on how it works and how docs
// can be customized.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "generate":
			if err := generate.Run(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "version":
			fmt.Printf("terraform-provider-typesense %s\n", version)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	// Default: run as Terraform provider
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/alanm/typesense",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}

func printUsage() {
	fmt.Printf(`terraform-provider-typesense %s

Usage:
  terraform-provider-typesense [command]

Commands:
  generate    Generate Terraform configuration from existing Typesense resources
  version     Print version information
  help        Show this help message

When run without a command, the provider starts in Terraform plugin mode.

For generate command help:
  terraform-provider-typesense generate --help
`, version)
}
