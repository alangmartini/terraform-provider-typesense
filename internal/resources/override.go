package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ resource.Resource = &OverrideResource{}
var _ resource.ResourceWithImportState = &OverrideResource{}

// NewOverrideResource creates a new override resource
func NewOverrideResource() resource.Resource {
	return &OverrideResource{}
}

// OverrideResource defines the resource implementation.
type OverrideResource struct {
	client *client.ServerClient
}

// OverrideResourceModel describes the resource data model.
type OverrideResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Collection          types.String `tfsdk:"collection"`
	Name                types.String `tfsdk:"name"`
	Rule                types.Object `tfsdk:"rule"`
	Includes            types.List   `tfsdk:"includes"`
	Excludes            types.List   `tfsdk:"excludes"`
	FilterBy            types.String `tfsdk:"filter_by"`
	SortBy              types.String `tfsdk:"sort_by"`
	ReplaceQuery        types.String `tfsdk:"replace_query"`
	RemoveMatchedTokens types.Bool   `tfsdk:"remove_matched_tokens"`
	FilterCuratedHits   types.Bool   `tfsdk:"filter_curated_hits"`
	EffectiveFromTs     types.Int64  `tfsdk:"effective_from_ts"`
	EffectiveToTs       types.Int64  `tfsdk:"effective_to_ts"`
	StopProcessing      types.Bool   `tfsdk:"stop_processing"`
}

// OverrideRuleModel describes the rule block
type OverrideRuleModel struct {
	Query types.String `tfsdk:"query"`
	Match types.String `tfsdk:"match"`
	Tags  types.List   `tfsdk:"tags"`
}

// OverrideIncludeModel describes an include block
type OverrideIncludeModel struct {
	ID       types.String `tfsdk:"id"`
	Position types.Int64  `tfsdk:"position"`
}

// OverrideExcludeModel describes an exclude block
type OverrideExcludeModel struct {
	ID types.String `tfsdk:"id"`
}

func (r *OverrideResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_override"
}

func (r *OverrideResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense override/curation rule for a collection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier (collection/name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"collection": schema.StringAttribute{
				Description: "The name of the collection this override belongs to.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name/ID of the override rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rule": schema.SingleNestedAttribute{
				Description: "The rule that triggers this override.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"query": schema.StringAttribute{
						Description: "The query pattern to match.",
						Optional:    true,
					},
					"match": schema.StringAttribute{
						Description: "Match type: 'exact' or 'contains'.",
						Optional:    true,
					},
					"tags": schema.ListAttribute{
						Description: "Tags to match for triggering the override.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"filter_by": schema.StringAttribute{
				Description: "Filter expression to apply.",
				Optional:    true,
			},
			"sort_by": schema.StringAttribute{
				Description: "Sort expression to apply.",
				Optional:    true,
			},
			"replace_query": schema.StringAttribute{
				Description: "Query to replace the original query with.",
				Optional:    true,
			},
			"remove_matched_tokens": schema.BoolAttribute{
				Description: "Remove matched tokens from the query.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"filter_curated_hits": schema.BoolAttribute{
				Description: "Apply filters to curated hits as well.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"effective_from_ts": schema.Int64Attribute{
				Description: "Unix timestamp from when this override is effective.",
				Optional:    true,
			},
			"effective_to_ts": schema.Int64Attribute{
				Description: "Unix timestamp until when this override is effective.",
				Optional:    true,
			},
			"stop_processing": schema.BoolAttribute{
				Description: "Stop processing further overrides if this one matches.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"includes": schema.ListNestedBlock{
				Description: "Documents to include/pin in results.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Document ID to include.",
							Required:    true,
						},
						"position": schema.Int64Attribute{
							Description: "Position to pin the document at (1-indexed).",
							Required:    true,
						},
					},
				},
			},
			"excludes": schema.ListNestedBlock{
				Description: "Documents to exclude from results.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Document ID to exclude.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *OverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage overrides.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *OverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	override, diags := r.modelToOverride(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateOverride(ctx, data.Collection.ValueString(), override)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create override: %s", err))
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", data.Collection.ValueString(), created.ID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	override, err := r.client.GetOverride(ctx, data.Collection.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read override: %s", err))
		return
	}

	if override == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromOverride(ctx, &data, override)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data OverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	override, diags := r.modelToOverride(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.CreateOverride(ctx, data.Collection.ValueString(), override)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update override: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOverride(ctx, data.Collection.ValueString(), data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete override: %s", err))
		return
	}
}

func (r *OverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: collection/override_name
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: collection/override_name, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

func (r *OverrideResource) modelToOverride(ctx context.Context, data *OverrideResourceModel) (*client.Override, diag.Diagnostics) {
	var diags diag.Diagnostics

	override := &client.Override{
		ID:                  data.Name.ValueString(),
		RemoveMatchedTokens: data.RemoveMatchedTokens.ValueBool(),
		FilterCuratedHits:   data.FilterCuratedHits.ValueBool(),
		StopProcessing:      data.StopProcessing.ValueBool(),
	}

	// Extract rule
	if !data.Rule.IsNull() {
		var rule OverrideRuleModel
		diags.Append(data.Rule.As(ctx, &rule, basetypes.ObjectAsOptions{})...)

		override.Rule = client.OverrideRule{
			Query: rule.Query.ValueString(),
			Match: rule.Match.ValueString(),
		}

		if !rule.Tags.IsNull() {
			var tags []string
			diags.Append(rule.Tags.ElementsAs(ctx, &tags, false)...)
			override.Rule.Tags = tags
		}
	}

	// Optional fields
	if !data.FilterBy.IsNull() {
		override.FilterBy = data.FilterBy.ValueString()
	}
	if !data.SortBy.IsNull() {
		override.SortBy = data.SortBy.ValueString()
	}
	if !data.ReplaceQuery.IsNull() {
		override.ReplaceQuery = data.ReplaceQuery.ValueString()
	}
	if !data.EffectiveFromTs.IsNull() {
		override.EffectiveFromTs = data.EffectiveFromTs.ValueInt64()
	}
	if !data.EffectiveToTs.IsNull() {
		override.EffectiveToTs = data.EffectiveToTs.ValueInt64()
	}

	// Extract includes
	if !data.Includes.IsNull() {
		var includes []OverrideIncludeModel
		diags.Append(data.Includes.ElementsAs(ctx, &includes, false)...)

		for _, inc := range includes {
			override.Includes = append(override.Includes, client.OverrideInclude{
				ID:       inc.ID.ValueString(),
				Position: int(inc.Position.ValueInt64()),
			})
		}
	}

	// Extract excludes
	if !data.Excludes.IsNull() {
		var excludes []OverrideExcludeModel
		diags.Append(data.Excludes.ElementsAs(ctx, &excludes, false)...)

		for _, exc := range excludes {
			override.Excludes = append(override.Excludes, client.OverrideExclude{
				ID: exc.ID.ValueString(),
			})
		}
	}

	return override, diags
}

func (r *OverrideResource) updateModelFromOverride(ctx context.Context, data *OverrideResourceModel, override *client.Override) {
	data.FilterBy = types.StringValue(override.FilterBy)
	data.SortBy = types.StringValue(override.SortBy)
	data.ReplaceQuery = types.StringValue(override.ReplaceQuery)
	data.RemoveMatchedTokens = types.BoolValue(override.RemoveMatchedTokens)
	data.FilterCuratedHits = types.BoolValue(override.FilterCuratedHits)
	data.StopProcessing = types.BoolValue(override.StopProcessing)

	if override.EffectiveFromTs > 0 {
		data.EffectiveFromTs = types.Int64Value(override.EffectiveFromTs)
	}
	if override.EffectiveToTs > 0 {
		data.EffectiveToTs = types.Int64Value(override.EffectiveToTs)
	}

	// Update rule
	ruleAttrTypes := map[string]attr.Type{
		"query": types.StringType,
		"match": types.StringType,
		"tags":  types.ListType{ElemType: types.StringType},
	}

	var tagsValue attr.Value
	if len(override.Rule.Tags) > 0 {
		tagValues := make([]types.String, len(override.Rule.Tags))
		for i, t := range override.Rule.Tags {
			tagValues[i] = types.StringValue(t)
		}
		tagsValue, _ = types.ListValueFrom(ctx, types.StringType, tagValues)
	} else {
		tagsValue = types.ListNull(types.StringType)
	}

	data.Rule, _ = types.ObjectValue(ruleAttrTypes, map[string]attr.Value{
		"query": types.StringValue(override.Rule.Query),
		"match": types.StringValue(override.Rule.Match),
		"tags":  tagsValue,
	})

	// Update includes
	if len(override.Includes) > 0 {
		includeAttrTypes := map[string]attr.Type{
			"id":       types.StringType,
			"position": types.Int64Type,
		}
		includeObjType := types.ObjectType{AttrTypes: includeAttrTypes}

		includeValues := make([]attr.Value, len(override.Includes))
		for i, inc := range override.Includes {
			includeValues[i], _ = types.ObjectValue(includeAttrTypes, map[string]attr.Value{
				"id":       types.StringValue(inc.ID),
				"position": types.Int64Value(int64(inc.Position)),
			})
		}
		data.Includes, _ = types.ListValue(includeObjType, includeValues)
	}

	// Update excludes
	if len(override.Excludes) > 0 {
		excludeAttrTypes := map[string]attr.Type{
			"id": types.StringType,
		}
		excludeObjType := types.ObjectType{AttrTypes: excludeAttrTypes}

		excludeValues := make([]attr.Value, len(override.Excludes))
		for i, exc := range override.Excludes {
			excludeValues[i], _ = types.ObjectValue(excludeAttrTypes, map[string]attr.Value{
				"id": types.StringValue(exc.ID),
			})
		}
		data.Excludes, _ = types.ListValue(excludeObjType, excludeValues)
	}
}
