package resources

import (
	"context"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &CollectionAliasResource{}
var _ resource.ResourceWithImportState = &CollectionAliasResource{}

// NewCollectionAliasResource creates a new collection alias resource
func NewCollectionAliasResource() resource.Resource {
	return &CollectionAliasResource{}
}

// CollectionAliasResource defines the resource implementation.
type CollectionAliasResource struct {
	client *client.ServerClient
}

// CollectionAliasResourceModel describes the resource data model.
type CollectionAliasResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	CollectionName types.String `tfsdk:"collection_name"`
}

func (r *CollectionAliasResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection_alias"
}

func (r *CollectionAliasResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense collection alias. Aliases allow you to refer to a collection by a virtual name, enabling zero-downtime reindexing and blue-green deployments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the alias (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the alias. This is what you use in API calls instead of the actual collection name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"collection_name": schema.StringAttribute{
				Description: "The name of the collection this alias points to.",
				Required:    true,
			},
		},
	}
}

func (r *CollectionAliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*providertypes.ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T.", req.ProviderData),
		)
		return
	}

	if providerData.ServerClient == nil {
		resp.Diagnostics.AddError(
			"Server API Not Configured",
			"The server_host and server_api_key must be configured in the provider to manage collection aliases.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *CollectionAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionAliasResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	alias := &client.CollectionAlias{
		Name:           data.Name.ValueString(),
		CollectionName: data.CollectionName.ValueString(),
	}

	created, err := r.client.UpsertCollectionAlias(ctx, alias)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create collection alias: %s", err))
		return
	}

	data.ID = types.StringValue(created.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	alias, err := r.client.GetCollectionAlias(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read collection alias: %s", err))
		return
	}

	if alias == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.CollectionName = types.StringValue(alias.CollectionName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionAliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CollectionAliasResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	alias := &client.CollectionAlias{
		Name:           data.Name.ValueString(),
		CollectionName: data.CollectionName.ValueString(),
	}

	_, err := r.client.UpsertCollectionAlias(ctx, alias)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update collection alias: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionAliasResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCollectionAlias(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete collection alias: %s", err))
		return
	}
}

func (r *CollectionAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
