package datasources

import (
	"context"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ServerInfoDataSource{}

// NewServerInfoDataSource creates a new server info data source
func NewServerInfoDataSource() datasource.DataSource {
	return &ServerInfoDataSource{}
}

// ServerInfoDataSource defines the data source implementation
type ServerInfoDataSource struct {
	client *client.ServerClient
}

// ServerInfoDataSourceModel describes the data source data model
type ServerInfoDataSourceModel struct {
	Version types.String `tfsdk:"version"`
	State   types.Int64  `tfsdk:"state"`
}

func (d *ServerInfoDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_info"
}

func (d *ServerInfoDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves version and state information from the Typesense server.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				Description: "The Typesense server version (e.g., \"30.1\").",
				Computed:    true,
			},
			"state": schema.Int64Attribute{
				Description: "The server state (e.g., 1 for ready).",
				Computed:    true,
			},
		},
	}
}

func (d *ServerInfoDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*providertypes.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	if providerData.ServerClient == nil {
		resp.Diagnostics.AddError(
			"Server API Not Configured",
			"The server_host and server_api_key must be configured in the provider to read server info.",
		)
		return
	}

	d.client = providerData.ServerClient
}

func (d *ServerInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ServerInfoDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	info, err := d.client.GetServerInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get server info: %s", err))
		return
	}

	data.Version = types.StringValue(info.Version)
	data.State = types.Int64Value(int64(info.State))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
