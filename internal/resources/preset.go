package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/alanm/terraform-provider-typesense/internal/version"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PresetResource{}
var _ resource.ResourceWithImportState = &PresetResource{}

// NewPresetResource creates a new preset resource
func NewPresetResource() resource.Resource {
	return &PresetResource{}
}

// PresetResource defines the resource implementation.
type PresetResource struct {
	client         *client.ServerClient
	featureChecker version.FeatureChecker
}

// PresetResourceModel describes the resource data model.
type PresetResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *PresetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_preset"
}

func (r *PresetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense search preset. Presets allow you to store search parameters server-side and reference them by name in queries.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the preset (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the preset. This is used to reference the preset in search queries via the 'preset' parameter.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "JSON-encoded search parameters for this preset. Can include any valid search parameters like q, query_by, filter_by, sort_by, facet_by, per_page, etc.",
				Required:    true,
			},
		},
	}
}

func (r *PresetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage presets.",
		)
		return
	}

	r.client = providerData.ServerClient
	r.featureChecker = providerData.FeatureChecker
}

func (r *PresetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if diags := version.CheckVersionRequirement(r.featureChecker, version.FeaturePresets, "typesense_preset"); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var data PresetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the JSON value
	var value map[string]any
	if err := json.Unmarshal([]byte(data.Value.ValueString()), &value); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("The value field must be valid JSON: %s", err))
		return
	}

	preset := &client.Preset{
		Name:  data.Name.ValueString(),
		Value: value,
	}

	created, err := r.client.UpsertPreset(ctx, preset)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create preset: %s", err))
		return
	}

	data.ID = types.StringValue(created.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PresetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	preset, err := r.client.GetPreset(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read preset: %s", err))
		return
	}

	if preset == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert value back to JSON string
	valueBytes, err := json.Marshal(preset.Value)
	if err != nil {
		resp.Diagnostics.AddError("Serialization Error", fmt.Sprintf("Unable to serialize preset value: %s", err))
		return
	}
	data.Value = types.StringValue(string(valueBytes))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PresetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the JSON value
	var value map[string]any
	if err := json.Unmarshal([]byte(data.Value.ValueString()), &value); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("The value field must be valid JSON: %s", err))
		return
	}

	preset := &client.Preset{
		Name:  data.Name.ValueString(),
		Value: value,
	}

	_, err := r.client.UpsertPreset(ctx, preset)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update preset: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PresetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PresetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePreset(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete preset: %s", err))
		return
	}
}

func (r *PresetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
