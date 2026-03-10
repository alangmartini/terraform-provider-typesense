package provider

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func metadataNamesFromResources(t *testing.T, resources []func() resource.Resource) []string {
	t.Helper()

	names := make([]string, 0, len(resources))
	for _, factory := range resources {
		r := factory()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{
			ProviderTypeName: tfnames.ProviderTypeName,
		}, &resp)
		names = append(names, resp.TypeName)
	}

	sort.Strings(names)
	return names
}

func metadataNamesFromDataSources(t *testing.T, dataSources []func() datasource.DataSource) []string {
	t.Helper()

	names := make([]string, 0, len(dataSources))
	for _, factory := range dataSources {
		d := factory()
		var resp datasource.MetadataResponse
		d.Metadata(context.Background(), datasource.MetadataRequest{
			ProviderTypeName: tfnames.ProviderTypeName,
		}, &resp)
		names = append(names, resp.TypeName)
	}

	sort.Strings(names)
	return names
}

func TestRegisteredResourceAndDataSourceTypeNamesMatchSharedRegistry(t *testing.T) {
	p := New("test")().(*TypesenseProvider)

	var providerMeta frameworkprovider.MetadataResponse
	p.Metadata(context.Background(), frameworkprovider.MetadataRequest{}, &providerMeta)
	if providerMeta.TypeName != tfnames.ProviderTypeName {
		t.Fatalf("provider type name = %q, want %q", providerMeta.TypeName, tfnames.ProviderTypeName)
	}

	resourceNames := metadataNamesFromResources(t, p.Resources(context.Background()))
	expectedResourceNames := make([]string, 0, len(tfnames.ResourceNames))
	for _, name := range tfnames.ResourceNames {
		expectedResourceNames = append(expectedResourceNames, tfnames.FullTypeName(name))
	}
	sort.Strings(expectedResourceNames)
	if len(resourceNames) != len(expectedResourceNames) {
		t.Fatalf("registered %d resource types, shared registry has %d", len(resourceNames), len(expectedResourceNames))
	}
	for i := range resourceNames {
		if resourceNames[i] != expectedResourceNames[i] {
			t.Fatalf("resource type mismatch at index %d: got %q want %q", i, resourceNames[i], expectedResourceNames[i])
		}
	}

	dataSourceNames := metadataNamesFromDataSources(t, p.DataSources(context.Background()))
	expectedDataSourceNames := make([]string, 0, len(tfnames.DataSourceNames))
	for _, name := range tfnames.DataSourceNames {
		expectedDataSourceNames = append(expectedDataSourceNames, tfnames.FullTypeName(name))
	}
	sort.Strings(expectedDataSourceNames)
	if len(dataSourceNames) != len(expectedDataSourceNames) {
		t.Fatalf("registered %d data source types, shared registry has %d", len(dataSourceNames), len(expectedDataSourceNames))
	}
	for i := range dataSourceNames {
		if dataSourceNames[i] != expectedDataSourceNames[i] {
			t.Fatalf("data source type mismatch at index %d: got %q want %q", i, dataSourceNames[i], expectedDataSourceNames[i])
		}
	}
}
