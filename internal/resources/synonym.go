package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SynonymResource{}
var _ resource.ResourceWithImportState = &SynonymResource{}

// NewSynonymResource creates a new synonym resource
func NewSynonymResource() resource.Resource {
	return &SynonymResource{}
}

// SynonymResource defines the resource implementation.
type SynonymResource struct {
	client *client.ServerClient
}

// SynonymResourceModel describes the resource data model.
type SynonymResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Collection types.String `tfsdk:"collection"`
	Name       types.String `tfsdk:"name"`
	Root       types.String `tfsdk:"root"`
	Synonyms   types.List   `tfsdk:"synonyms"`
}

func (r *SynonymResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synonym"
}

func (r *SynonymResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense synonym configuration for a collection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier (collection/name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"collection": schema.StringAttribute{
				Description: "The name of the collection this synonym belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name/ID of the synonym rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"root": schema.StringAttribute{
				Description: "For one-way synonyms, the root word that the synonyms map to. Leave empty for multi-way synonyms.",
				Optional:    true,
			},
			"synonyms": schema.ListAttribute{
				Description: "List of synonym words.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *SynonymResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage synonyms.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *SynonymResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var synonyms []string
	resp.Diagnostics.Append(data.Synonyms.ElementsAs(ctx, &synonyms, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	synonym := &client.Synonym{
		ID:       data.Name.ValueString(),
		Synonyms: synonyms,
	}

	if !data.Root.IsNull() {
		synonym.Root = data.Root.ValueString()
	}

	created, err := r.client.CreateSynonym(ctx, data.Collection.ValueString(), synonym)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create synonym: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Collection.ValueString(), created.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	synonym, err := r.client.GetSynonym(ctx, data.Collection.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read synonym: %s", err))
		return
	}

	if synonym == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update synonyms list
	synonymValues := make([]types.String, len(synonym.Synonyms))
	for i, s := range synonym.Synonyms {
		synonymValues[i] = types.StringValue(s)
	}
	data.Synonyms, _ = types.ListValueFrom(ctx, types.StringType, synonymValues)

	if synonym.Root != "" {
		data.Root = types.StringValue(synonym.Root)
	} else {
		data.Root = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var synonyms []string
	resp.Diagnostics.Append(data.Synonyms.ElementsAs(ctx, &synonyms, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	synonym := &client.Synonym{
		ID:       data.Name.ValueString(),
		Synonyms: synonyms,
	}

	if !data.Root.IsNull() {
		synonym.Root = data.Root.ValueString()
	}

	_, err := r.client.CreateSynonym(ctx, data.Collection.ValueString(), synonym)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update synonym: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSynonym(ctx, data.Collection.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete synonym: %s", err))
		return
	}
}

func (r *SynonymResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: collection/synonym_name
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: collection/synonym_name, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
