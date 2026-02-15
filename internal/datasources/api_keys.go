package datasources

import (
	"context"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &APIKeysDataSource{}

// NewAPIKeysDataSource creates a new API keys data source
func NewAPIKeysDataSource() datasource.DataSource {
	return &APIKeysDataSource{}
}

// APIKeysDataSource defines the data source implementation
type APIKeysDataSource struct {
	client *client.ServerClient
}

// APIKeysDataSourceModel describes the data source data model
type APIKeysDataSourceModel struct {
	Keys types.List `tfsdk:"keys"`
}

func (d *APIKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_keys"
}

func (d *APIKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all API keys on the Typesense server. Note: the API only returns key value prefixes, not full key values.",
		Attributes: map[string]schema.Attribute{
			"keys": schema.ListNestedAttribute{
				Description: "List of API keys.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Numeric ID of the API key.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the API key.",
							Computed:    true,
						},
						"actions": schema.ListAttribute{
							Description: "List of allowed actions.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"collections": schema.ListAttribute{
							Description: "List of collections this key can access.",
							Computed:    true,
							ElementType: types.StringType,
						},
						"value_prefix": schema.StringAttribute{
							Description: "Prefix of the API key value (full value is not returned by the API).",
							Computed:    true,
						},
						"expires_at": schema.Int64Attribute{
							Description: "Unix timestamp when the key expires. 0 means no expiration.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *APIKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to read API keys.",
		)
		return
	}

	d.client = providerData.ServerClient
}

func (d *APIKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data APIKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, err := d.client.ListAPIKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list API keys: %s", err))
		return
	}

	keyAttrTypes := map[string]attr.Type{
		"id":           types.Int64Type,
		"description":  types.StringType,
		"actions":      types.ListType{ElemType: types.StringType},
		"collections":  types.ListType{ElemType: types.StringType},
		"value_prefix": types.StringType,
		"expires_at":   types.Int64Type,
	}

	keyValues := make([]attr.Value, len(keys))
	for i, k := range keys {
		actions, _ := types.ListValueFrom(ctx, types.StringType, k.Actions)
		collections, _ := types.ListValueFrom(ctx, types.StringType, k.Collections)

		keyValues[i], _ = types.ObjectValue(keyAttrTypes, map[string]attr.Value{
			"id":           types.Int64Value(k.ID),
			"description":  types.StringValue(k.Description),
			"actions":      actions,
			"collections":  collections,
			"value_prefix": types.StringValue(k.Value),
			"expires_at":   types.Int64Value(k.ExpiresAt),
		})
	}

	keyObjType := types.ObjectType{AttrTypes: keyAttrTypes}
	data.Keys, _ = types.ListValue(keyObjType, keyValues)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
