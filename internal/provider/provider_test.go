package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
//
//nolint:unused // Scaffolding for future acceptance tests
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"typesense": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that the required environment variables are set
// for acceptance testing.
//
//nolint:unused // Scaffolding for future acceptance tests
func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("TYPESENSE_HOST"); v == "" {
		t.Fatal("TYPESENSE_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("TYPESENSE_API_KEY"); v == "" {
		t.Fatal("TYPESENSE_API_KEY must be set for acceptance tests")
	}
}
