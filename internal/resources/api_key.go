package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &APIKeyResource{}
var _ resource.ResourceWithImportState = &APIKeyResource{}

// NewAPIKeyResource creates a new API key resource
func NewAPIKeyResource() resource.Resource {
	return &APIKeyResource{}
}

// APIKeyResource defines the resource implementation.
type APIKeyResource struct {
	client *client.ServerClient
}

// APIKeyResourceModel describes the resource data model.
type APIKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Value       types.String `tfsdk:"value"`
	ValuePrefix types.String `tfsdk:"value_prefix"`
	Description types.String `tfsdk:"description"`
	Actions     types.List   `tfsdk:"actions"`
	Collections types.List   `tfsdk:"collections"`
	ExpiresAt   types.Int64  `tfsdk:"expires_at"`
	AutoDelete  types.Bool   `tfsdk:"autodelete"`
}

func (r *APIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense API key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the API key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"value": schema.StringAttribute{
				Description: "The API key value. Set this to use a specific key value (e.g., for consistent keys across environments). If omitted, Typesense generates one automatically. Only the full value is available at creation time; subsequent reads return only a 4-character prefix.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value_prefix": schema.StringAttribute{
				Description: "First 4 characters of the API key value, useful for identifying keys.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the API key.",
				Optional:    true,
			},
			"actions": schema.ListAttribute{
				Description: "List of actions this key can perform (e.g., 'documents:search', 'documents:get', 'collections:create', '*').",
				Required:    true,
				ElementType: types.StringType,
			},
			"collections": schema.ListAttribute{
				Description: "List of collections this key has access to. Use '*' for all collections.",
				Required:    true,
				ElementType: types.StringType,
			},
			"expires_at": schema.Int64Attribute{
				Description: "Unix timestamp when this key expires. 0 means never expires.",
				Optional:    true,
			},
			"autodelete": schema.BoolAttribute{
				Description: "If true, the API key is automatically deleted after it expires. Requires expires_at to be set.",
				Optional:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *APIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage API keys.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var actions []string
	resp.Diagnostics.Append(data.Actions.ElementsAs(ctx, &actions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var collections []string
	resp.Diagnostics.Append(data.Collections.ElementsAs(ctx, &collections, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := &client.APIKey{
		Actions:     actions,
		Collections: collections,
	}

	if !data.Description.IsNull() {
		apiKey.Description = data.Description.ValueString()
	}

	if !data.ExpiresAt.IsNull() {
		apiKey.ExpiresAt = data.ExpiresAt.ValueInt64()
	}

	if !data.Value.IsNull() && !data.Value.IsUnknown() {
		apiKey.Value = data.Value.ValueString()
	}

	if !data.AutoDelete.IsNull() {
		apiKey.AutoDelete = data.AutoDelete.ValueBool()
	}

	created, err := r.client.CreateAPIKey(ctx, apiKey)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create API key: %s", err))
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(created.ID, 10))
	data.Value = types.StringValue(created.Value)

	// Compute value_prefix from the full key value
	prefix := created.Value
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}
	data.ValuePrefix = types.StringValue(prefix)

	// Also update expires_at from the response if it was set in the config
	// This ensures consistency between what was requested and what the API stored
	if !data.ExpiresAt.IsNull() && created.ExpiresAt > 0 {
		data.ExpiresAt = types.Int64Value(created.ExpiresAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse API key ID: %s", err))
		return
	}

	apiKey, err := r.client.GetAPIKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read API key: %s", err))
		return
	}

	if apiKey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update from API response (note: full value is not returned on read, only a prefix)
	if apiKey.Description != "" {
		data.Description = types.StringValue(apiKey.Description)
	}

	// Update value_prefix from the API response (GET returns only the first 4 chars in `value`)
	if apiKey.Value != "" {
		data.ValuePrefix = types.StringValue(apiKey.Value)
	}

	// Note: data.Value is preserved from state (UseStateForUnknown plan modifier)
	// Note: data.AutoDelete is preserved from state (not returned by GET API)

	// Update actions
	actionValues := make([]types.String, len(apiKey.Actions))
	for i, a := range apiKey.Actions {
		actionValues[i] = types.StringValue(a)
	}
	data.Actions, _ = types.ListValueFrom(ctx, types.StringType, actionValues)

	// Update collections
	collectionValues := make([]types.String, len(apiKey.Collections))
	for i, c := range apiKey.Collections {
		collectionValues[i] = types.StringValue(c)
	}
	data.Collections, _ = types.ListValueFrom(ctx, types.StringType, collectionValues)

	// Update expires_at from API response if present and not the far-future default
	// Typesense returns 64723363199 (year 4022) as default when not explicitly set
	// We only store it in state if it was explicitly set by the user
	if apiKey.ExpiresAt > 0 && apiKey.ExpiresAt < 32503680000 {
		// This is a real expiration date (before year 3000), store it
		data.ExpiresAt = types.Int64Value(apiKey.ExpiresAt)
	} else if !data.ExpiresAt.IsNull() {
		// expires_at was previously set in state, update it even if it's a far-future value
		data.ExpiresAt = types.Int64Value(apiKey.ExpiresAt)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *APIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// API keys cannot be updated after creation
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"API keys cannot be updated after creation. Delete and recreate the key to make changes.",
	)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data APIKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Parse Error", fmt.Sprintf("Unable to parse API key ID: %s", err))
		return
	}

	err = r.client.DeleteAPIKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete API key: %s", err))
		return
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
