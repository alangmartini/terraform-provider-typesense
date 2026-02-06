package resources

import (
	"context"
	"encoding/json"
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

var _ resource.Resource = &AnalyticsRuleResource{}
var _ resource.ResourceWithImportState = &AnalyticsRuleResource{}

// NewAnalyticsRuleResource creates a new analytics rule resource
func NewAnalyticsRuleResource() resource.Resource {
	return &AnalyticsRuleResource{}
}

// AnalyticsRuleResource defines the resource implementation.
type AnalyticsRuleResource struct {
	client *client.ServerClient
}

// AnalyticsRuleResourceModel describes the resource data model.
type AnalyticsRuleResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Type      types.String `tfsdk:"type"`
	EventType types.String `tfsdk:"event_type"`
	Params    types.String `tfsdk:"params"`
}

func (r *AnalyticsRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analytics_rule"
}

func (r *AnalyticsRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense analytics rule. Analytics rules aggregate search queries and user events for query suggestions, popularity scoring, and identifying content gaps.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the analytics rule (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the analytics rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of analytics rule: 'popular_queries' (track frequent searches), 'nohits_queries' (track zero-result searches), or 'counter' (increment popularity based on events).",
				Required:    true,
			},
			"event_type": schema.StringAttribute{
				Description: "The event type this rule tracks: 'search' for query-based rules (popular_queries, nohits_queries), or 'click'/'conversion'/'visit' for counter rules.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"params": schema.StringAttribute{
				Description: "JSON-encoded parameters for the analytics rule. Structure varies by type but typically includes 'source' (collections and events to monitor) and 'destination' (where to store aggregated data).",
				Required:    true,
			},
		},
	}
}

func (r *AnalyticsRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage analytics rules.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *AnalyticsRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AnalyticsRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the JSON params
	var params map[string]any
	if err := json.Unmarshal([]byte(data.Params.ValueString()), &params); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("The params field must be valid JSON: %s", err))
		return
	}

	rule := &client.AnalyticsRule{
		Name:      data.Name.ValueString(),
		Type:      data.Type.ValueString(),
		EventType: data.EventType.ValueString(),
		Params:    params,
	}

	created, err := r.client.UpsertAnalyticsRule(ctx, rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create analytics rule: %s", err))
		return
	}

	data.ID = types.StringValue(created.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AnalyticsRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetAnalyticsRule(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read analytics rule: %s", err))
		return
	}

	if rule == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Type = types.StringValue(rule.Type)

	// event_type is not returned by the Typesense API.
	// For imports (when event_type is null), infer it from the rule type.
	// For refreshes, preserve the existing state value.
	if data.EventType.IsNull() || data.EventType.ValueString() == "" {
		// Infer event_type based on rule type
		switch rule.Type {
		case "popular_queries", "nohits_queries":
			data.EventType = types.StringValue("search")
		case "counter":
			// For counter rules, try to extract from params.source.events
			if source, ok := rule.Params["source"].(map[string]any); ok {
				if events, ok := source["events"].([]any); ok && len(events) > 0 {
					if event, ok := events[0].(map[string]any); ok {
						if eventType, ok := event["type"].(string); ok {
							data.EventType = types.StringValue(eventType)
						}
					}
				}
			}
			// Default to "click" if we couldn't extract it
			if data.EventType.IsNull() || data.EventType.ValueString() == "" {
				data.EventType = types.StringValue("click")
			}
		default:
			data.EventType = types.StringValue("search")
		}
	}

	// For imports (when params is null), populate from API response.
	// For refreshes, preserve the user's original params to avoid drift
	// from server-side defaults (like expand_query, limit).
	if data.Params.IsNull() || data.Params.ValueString() == "" {
		paramsBytes, err := json.Marshal(rule.Params)
		if err != nil {
			resp.Diagnostics.AddError("Serialization Error", fmt.Sprintf("Unable to serialize analytics rule params: %s", err))
			return
		}
		data.Params = types.StringValue(string(paramsBytes))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AnalyticsRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the JSON params
	var params map[string]any
	if err := json.Unmarshal([]byte(data.Params.ValueString()), &params); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", fmt.Sprintf("The params field must be valid JSON: %s", err))
		return
	}

	rule := &client.AnalyticsRule{
		Name:      data.Name.ValueString(),
		Type:      data.Type.ValueString(),
		EventType: data.EventType.ValueString(),
		Params:    params,
	}

	_, err := r.client.UpsertAnalyticsRule(ctx, rule)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update analytics rule: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnalyticsRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AnalyticsRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAnalyticsRule(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete analytics rule: %s", err))
		return
	}
}

func (r *AnalyticsRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
