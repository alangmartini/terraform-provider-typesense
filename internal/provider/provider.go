// Package provider implements the Typesense Terraform provider
package provider

import (
	"context"
	"os"
	"strconv"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/resources"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/alanm/terraform-provider-typesense/internal/version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure TypesenseProvider satisfies various provider interfaces.
var _ provider.Provider = &TypesenseProvider{}

// TypesenseProvider defines the provider implementation.
type TypesenseProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TypesenseProviderModel describes the provider data model.
type TypesenseProviderModel struct {
	// Cloud Management API configuration
	CloudManagementAPIKey types.String `tfsdk:"cloud_management_api_key"`

	// Server API configuration
	ServerHost     types.String `tfsdk:"server_host"`
	ServerAPIKey   types.String `tfsdk:"server_api_key"`
	ServerPort     types.Int64  `tfsdk:"server_port"`
	ServerProtocol types.String `tfsdk:"server_protocol"`
}

// ProviderData is an alias for the shared type
type ProviderData = providertypes.ProviderData

func (p *TypesenseProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "typesense"
	resp.Version = p.version
}

func (p *TypesenseProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Typesense provider allows you to manage Typesense Cloud clusters and server resources like collections, synonyms, overrides, stopwords, and API keys.",
		Attributes: map[string]schema.Attribute{
			"cloud_management_api_key": schema.StringAttribute{
				Description: "API key for Typesense Cloud Management API. Can also be set via TYPESENSE_CLOUD_MANAGEMENT_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"server_host": schema.StringAttribute{
				Description: "Hostname of the Typesense server (e.g., 'xxx.a1.typesense.net' or 'localhost'). Can also be set via TYPESENSE_HOST environment variable.",
				Optional:    true,
			},
			"server_api_key": schema.StringAttribute{
				Description: "API key for Typesense Server API. Can also be set via TYPESENSE_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"server_port": schema.Int64Attribute{
				Description: "Port number for the Typesense server. Defaults to 443. Can also be set via TYPESENSE_PORT environment variable.",
				Optional:    true,
			},
			"server_protocol": schema.StringAttribute{
				Description: "Protocol for connecting to Typesense server ('http' or 'https'). Defaults to 'https'. Can also be set via TYPESENSE_PROTOCOL environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *TypesenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TypesenseProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get values from config or environment variables
	cloudAPIKey := getStringValue(config.CloudManagementAPIKey, "TYPESENSE_CLOUD_MANAGEMENT_API_KEY")
	serverHost := getStringValue(config.ServerHost, "TYPESENSE_HOST")
	serverAPIKey := getStringValue(config.ServerAPIKey, "TYPESENSE_API_KEY")
	serverPort := getInt64Value(config.ServerPort, "TYPESENSE_PORT", 443)
	serverProtocol := getStringValueWithDefault(config.ServerProtocol, "TYPESENSE_PROTOCOL", "https")

	providerData := &providertypes.ProviderData{}

	// Configure Cloud client if API key is provided
	if cloudAPIKey != "" {
		providerData.CloudClient = client.NewCloudClient(cloudAPIKey)
	}

	// Configure Server client if host and API key are provided
	if serverHost != "" && serverAPIKey != "" {
		providerData.ServerClient = client.NewServerClient(serverHost, serverAPIKey, int(serverPort), serverProtocol)

		// Detect server version for feature-aware API selection
		serverVersion, featureChecker, versionDiag := detectServerVersion(ctx, providerData.ServerClient)
		if versionDiag != nil {
			resp.Diagnostics.Append(versionDiag)
		}
		providerData.ServerVersion = serverVersion
		providerData.FeatureChecker = featureChecker
	} else {
		// No server client, use fallback feature checker
		providerData.FeatureChecker = version.NewFallbackFeatureChecker()
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *TypesenseProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewClusterResource,
		resources.NewClusterConfigChangeResource,
		resources.NewCollectionResource,
		resources.NewSynonymResource,
		resources.NewOverrideResource,
		resources.NewStopwordsSetResource,
		resources.NewAPIKeyResource,
		resources.NewNLSearchModelResource,
	}
}

func (p *TypesenseProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// New creates a new provider instance
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TypesenseProvider{
			version: version,
		}
	}
}

// Helper functions for getting configuration values

func getStringValue(tfValue types.String, envVar string) string {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueString()
	}
	return os.Getenv(envVar)
}

func getStringValueWithDefault(tfValue types.String, envVar, defaultValue string) string {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueString()
	}
	if val := os.Getenv(envVar); val != "" {
		return val
	}
	return defaultValue
}

func getInt64Value(tfValue types.Int64, envVar string, defaultValue int64) int64 {
	if !tfValue.IsNull() && !tfValue.IsUnknown() {
		return tfValue.ValueInt64()
	}
	if val := os.Getenv(envVar); val != "" {
		if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// detectServerVersion queries the server for version information and creates
// an appropriate FeatureChecker. On failure, it returns a warning diagnostic
// and a FallbackFeatureChecker that allows runtime detection via 404 handling.
func detectServerVersion(ctx context.Context, serverClient *client.ServerClient) (*version.Version, version.FeatureChecker, diag.Diagnostic) {
	info, err := serverClient.GetServerInfo(ctx)
	if err != nil {
		// Version detection failed - use fallback checker
		// This is a warning, not an error, because resources can still
		// fall back to runtime detection via 404 handling
		return nil, version.NewFallbackFeatureChecker(), diag.NewWarningDiagnostic(
			"Could not detect Typesense server version",
			"Failed to retrieve server version information. The provider will use "+
				"runtime detection for version-specific features. Error: "+err.Error(),
		)
	}

	serverVersion, err := version.Parse(info.Version)
	if err != nil {
		// Version parsing failed - use fallback checker
		return nil, version.NewFallbackFeatureChecker(), diag.NewWarningDiagnostic(
			"Could not parse Typesense server version",
			"The server returned an unexpected version format: "+info.Version+". "+
				"The provider will use runtime detection for version-specific features. "+
				"Error: "+err.Error(),
		)
	}

	// Successfully detected version - create proper feature checker
	return serverVersion, version.NewFeatureChecker(serverVersion), nil
}
