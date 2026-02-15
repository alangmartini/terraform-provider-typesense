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

var _ datasource.DataSource = &CollectionsDataSource{}

// NewCollectionsDataSource creates a new collections data source
func NewCollectionsDataSource() datasource.DataSource {
	return &CollectionsDataSource{}
}

// CollectionsDataSource defines the data source implementation
type CollectionsDataSource struct {
	client *client.ServerClient
}

// CollectionsDataSourceModel describes the data source data model
type CollectionsDataSourceModel struct {
	Collections types.List `tfsdk:"collections"`
}

func (d *CollectionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collections"
}

func (d *CollectionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all collections on the Typesense server.",
		Attributes: map[string]schema.Attribute{
			"collections": schema.ListNestedAttribute{
				Description: "List of collections.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the collection.",
							Computed:    true,
						},
						"num_documents": schema.Int64Attribute{
							Description: "Number of documents in the collection.",
							Computed:    true,
						},
						"created_at": schema.Int64Attribute{
							Description: "Timestamp when the collection was created.",
							Computed:    true,
						},
						"default_sorting_field": schema.StringAttribute{
							Description: "The default field to sort results by.",
							Computed:    true,
						},
						"enable_nested_fields": schema.BoolAttribute{
							Description: "Whether nested fields support is enabled.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *CollectionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to read collections.",
		)
		return
	}

	d.client = providerData.ServerClient
}

func (d *CollectionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CollectionsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collections, err := d.client.ListCollections(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list collections: %s", err))
		return
	}

	collectionAttrTypes := map[string]attr.Type{
		"name":                  types.StringType,
		"num_documents":         types.Int64Type,
		"created_at":            types.Int64Type,
		"default_sorting_field": types.StringType,
		"enable_nested_fields":  types.BoolType,
	}

	collectionValues := make([]attr.Value, len(collections))
	for i, c := range collections {
		defaultSortingField := types.StringValue("")
		if c.DefaultSortingField != "" {
			defaultSortingField = types.StringValue(c.DefaultSortingField)
		}

		collectionValues[i], _ = types.ObjectValue(collectionAttrTypes, map[string]attr.Value{
			"name":                  types.StringValue(c.Name),
			"num_documents":         types.Int64Value(c.NumDocuments),
			"created_at":            types.Int64Value(c.CreatedAt),
			"default_sorting_field": defaultSortingField,
			"enable_nested_fields":  types.BoolValue(c.EnableNestedFields),
		})
	}

	collectionObjType := types.ObjectType{AttrTypes: collectionAttrTypes}
	data.Collections, _ = types.ListValue(collectionObjType, collectionValues)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
