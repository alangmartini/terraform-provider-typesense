// Package resources implements Terraform resources for Typesense
package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ClusterResource{}
var _ resource.ResourceWithImportState = &ClusterResource{}
var _ resource.ResourceWithModifyPlan = &ClusterResource{}

// NewClusterResource creates a new cluster resource
func NewClusterResource() resource.Resource {
	return &ClusterResource{}
}

// ClusterResource defines the resource implementation.
type ClusterResource struct {
	client *client.CloudClient
}

// ClusterResourceModel describes the resource data model.
type ClusterResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Memory                 types.String `tfsdk:"memory"`
	VCPU                   types.String `tfsdk:"vcpu"`
	HighAvailability       types.String `tfsdk:"high_availability"`
	SearchDeliveryNetwork  types.String `tfsdk:"search_delivery_network"`
	TypesenseServerVersion types.String `tfsdk:"typesense_server_version"`
	Regions                types.List   `tfsdk:"regions"`
	Status                 types.String `tfsdk:"status"`
	LoadBalancedHostname   types.String `tfsdk:"load_balanced_hostname"`
	Nodes                  types.List   `tfsdk:"nodes"`
	AdminAPIKey            types.String `tfsdk:"admin_api_key"`
	SearchAPIKey           types.String `tfsdk:"search_api_key"`
	AutoUpgradeCapacity    types.Bool   `tfsdk:"auto_upgrade_capacity"`
	CreatedAt              types.String `tfsdk:"created_at"`
}

func (r *ClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = tfnames.TypeName(req.ProviderTypeName, tfnames.ResourceCluster)
}

func (r *ClusterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense Cloud cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the cluster.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the cluster. This can be updated in place via the Cloud cluster update API.",
				Required:    true,
			},
			"memory": schema.StringAttribute{
				Description: "Memory configuration (e.g., '0.5_gb', '1_gb', '2_gb', '4_gb', '8_gb', '16_gb', '32_gb', '64_gb', '128_gb', '192_gb', '256_gb', '384_gb', '512_gb'). On existing clusters this is applied via the Cloud configuration changes API.",
				Required:    true,
			},
			"vcpu": schema.StringAttribute{
				Description: "vCPU configuration (e.g., '2_vcpus_4_hr_burst_per_day', '2_vcpus', '4_vcpus', '8_vcpus', etc.). On existing clusters this is applied via the Cloud configuration changes API.",
				Required:    true,
			},
			"high_availability": schema.StringAttribute{
				Description: "High availability setting ('yes', 'no', or 'yes_3_way', 'yes_5_way'). On existing clusters this is applied via the Cloud configuration changes API. Typesense Cloud documentation notes that once high availability is enabled, it cannot be turned off without recreating the cluster.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("no"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(
						func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
							if clusterHighAvailabilityReplacementNeeded(req.StateValue, req.PlanValue) {
								resp.RequiresReplace = true
							}
						},
						"Recreate cluster when disabling high availability",
						"Typesense Cloud does not allow disabling high availability on an existing cluster; recreate the cluster instead.",
					),
				},
			},
			"search_delivery_network": schema.StringAttribute{
				Description: "Search delivery network setting ('off', 'on'). This is set only at cluster creation time; changing it recreates the cluster.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("off"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"typesense_server_version": schema.StringAttribute{
				Description: "Typesense server version (e.g., '27.1', '26.0'). On existing clusters this is applied via the Cloud configuration changes API.",
				Required:    true,
			},
			"regions": schema.ListAttribute{
				Description: "List of regions to deploy the cluster in. This is set only at cluster creation time; changing it recreates the cluster.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the cluster.",
				Computed:    true,
			},
			"load_balanced_hostname": schema.StringAttribute{
				Description: "Load balanced hostname for the cluster.",
				Computed:    true,
			},
			"nodes": schema.ListAttribute{
				Description: "List of node hostnames.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"admin_api_key": schema.StringAttribute{
				Description: "Admin API key for the cluster.",
				Computed:    true,
				Sensitive:   true,
			},
			"search_api_key": schema.StringAttribute{
				Description: "Search-only API key for the cluster.",
				Computed:    true,
				Sensitive:   true,
			},
			"auto_upgrade_capacity": schema.BoolAttribute{
				Description: "Whether to auto-upgrade cluster capacity. This can be updated in place via the Cloud cluster update API.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				Description: "Timestamp when the cluster was created.",
				Computed:    true,
			},
		},
	}
}

func (r *ClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	providerData, ok := req.ProviderData.(*providertypes.ProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providertypes.ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	if providerData.CloudClient == nil {
		resp.Diagnostics.AddError(
			"Cloud Management API Not Configured",
			"The cloud_management_api_key must be configured in the provider to manage clusters.",
		)
		return
	}

	r.client = providerData.CloudClient
}

func (r *ClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state ClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	var plan ClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for _, warning := range clusterPlanWarnings(state, plan) {
		resp.Diagnostics.AddAttributeWarning(path.Root(warning.Attribute), warning.Summary, warning.Detail)
	}
}

func (r *ClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert regions from types.List to []string
	var regions []string
	resp.Diagnostics.Append(data.Regions.ElementsAs(ctx, &regions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cluster := &client.Cluster{
		Name:                   data.Name.ValueString(),
		Memory:                 data.Memory.ValueString(),
		VCPU:                   data.VCPU.ValueString(),
		HighAvailability:       data.HighAvailability.ValueString(),
		SearchDeliveryNetwork:  data.SearchDeliveryNetwork.ValueString(),
		TypesenseServerVersion: data.TypesenseServerVersion.ValueString(),
		Regions:                regions,
		AutoUpgradeCapacity:    data.AutoUpgradeCapacity.ValueBool(),
	}

	created, err := r.client.CreateCluster(ctx, cluster)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster: %s", err))
		return
	}

	// Preserve API keys from creation response (GetCluster doesn't return them)
	apiKeys := created.APIKeys

	// Wait for cluster to be ready (up to 15 minutes)
	ready, err := r.client.WaitForClusterReady(ctx, created.ID, 15*time.Minute)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for cluster to be ready: %s", err))
		return
	}

	// Restore API keys since GetCluster doesn't return them
	ready.APIKeys = apiKeys

	r.updateModelFromCluster(&data, ready)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve API keys from state (GetCluster doesn't return them)
	adminAPIKey := data.AdminAPIKey
	searchAPIKey := data.SearchAPIKey

	cluster, err := r.client.GetCluster(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster: %s", err))
		return
	}

	if cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromCluster(&data, cluster)

	// Restore API keys from state since GetCluster doesn't return them.
	// If keys were never available (e.g., imported cluster), set to empty string.
	if !adminAPIKey.IsNull() {
		data.AdminAPIKey = adminAPIKey
	} else if data.AdminAPIKey.IsNull() || data.AdminAPIKey.IsUnknown() {
		data.AdminAPIKey = types.StringValue("")
	}
	if !searchAPIKey.IsNull() {
		data.SearchAPIKey = searchAPIKey
	} else if data.SearchAPIKey.IsNull() || data.SearchAPIKey.IsUnknown() {
		data.SearchAPIKey = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ClusterResourceModel
	var state ClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	clusterID := data.ID.ValueString()

	// Step 1: Apply direct updates (name, auto_upgrade_capacity) — fast metadata changes
	if data.Name.ValueString() != state.Name.ValueString() ||
		data.AutoUpgradeCapacity.ValueBool() != state.AutoUpgradeCapacity.ValueBool() {
		cluster := &client.Cluster{
			Name:                data.Name.ValueString(),
			AutoUpgradeCapacity: data.AutoUpgradeCapacity.ValueBool(),
		}

		_, err := r.client.UpdateCluster(ctx, clusterID, cluster)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update cluster: %s", err))
			return
		}
	}

	// Step 2: Detect infrastructure changes that require the config change API
	configChange := &client.ClusterConfigChange{
		ClusterID: clusterID,
	}
	needsConfigChange := false

	if data.Memory.ValueString() != state.Memory.ValueString() {
		configChange.NewMemory = data.Memory.ValueString()
		needsConfigChange = true
	}
	if data.VCPU.ValueString() != state.VCPU.ValueString() {
		configChange.NewVCPU = data.VCPU.ValueString()
		needsConfigChange = true
	}
	if data.HighAvailability.ValueString() != state.HighAvailability.ValueString() {
		configChange.NewHighAvailability = data.HighAvailability.ValueString()
		needsConfigChange = true
	}
	if data.TypesenseServerVersion.ValueString() != state.TypesenseServerVersion.ValueString() {
		configChange.NewTypesenseVersion = data.TypesenseServerVersion.ValueString()
		needsConfigChange = true
	}

	if needsConfigChange {
		// Set perform_change_at to now for immediate execution
		configChange.PerformChangeAt = time.Now().Unix()

		_, err := r.client.CreateClusterConfigChange(ctx, configChange)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster configuration change: %s", err))
			return
		}

		// Wait for the cluster to finish applying the config change (up to 15 minutes)
		_, err = r.client.WaitForClusterReady(ctx, clusterID, 15*time.Minute)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Error waiting for cluster configuration change to complete: %s", err))
			return
		}
	}

	// Step 3: Refresh state from the API to capture the final cluster state
	refreshed, err := r.client.GetCluster(ctx, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster after update: %s", err))
		return
	}

	r.updateModelFromCluster(&data, refreshed)

	// Restore API keys from state since GetCluster doesn't return them.
	// If keys were never available (e.g., imported cluster), set to empty string
	// to avoid "unknown value after apply" errors.
	if !state.AdminAPIKey.IsNull() {
		data.AdminAPIKey = state.AdminAPIKey
	} else if data.AdminAPIKey.IsNull() || data.AdminAPIKey.IsUnknown() {
		data.AdminAPIKey = types.StringValue("")
	}
	if !state.SearchAPIKey.IsNull() {
		data.SearchAPIKey = state.SearchAPIKey
	} else if data.SearchAPIKey.IsNull() || data.SearchAPIKey.IsUnknown() {
		data.SearchAPIKey = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCluster(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster: %s", err))
		return
	}
}

func (r *ClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ClusterResource) updateModelFromCluster(data *ClusterResourceModel, cluster *client.Cluster) {
	data.ID = types.StringValue(cluster.ID)
	data.Name = types.StringValue(cluster.Name)
	data.Memory = types.StringValue(cluster.Memory)
	data.VCPU = types.StringValue(cluster.VCPU)
	data.HighAvailability = types.StringValue(cluster.HighAvailability)
	data.SearchDeliveryNetwork = types.StringValue(cluster.SearchDeliveryNetwork)
	data.TypesenseServerVersion = types.StringValue(cluster.TypesenseServerVersion)
	data.Status = types.StringValue(cluster.Status)
	data.LoadBalancedHostname = types.StringValue(cluster.Hostnames.LoadBalanced)
	data.AutoUpgradeCapacity = types.BoolValue(cluster.AutoUpgradeCapacity)
	data.CreatedAt = types.StringValue(cluster.CreatedAt)

	// Convert regions
	regionValues := make([]types.String, len(cluster.Regions))
	for i, r := range cluster.Regions {
		regionValues[i] = types.StringValue(r)
	}
	data.Regions, _ = types.ListValueFrom(context.Background(), types.StringType, regionValues)

	// Convert nodes
	nodeValues := make([]types.String, len(cluster.Hostnames.Nodes))
	for i, n := range cluster.Hostnames.Nodes {
		nodeValues[i] = types.StringValue(n)
	}
	data.Nodes, _ = types.ListValueFrom(context.Background(), types.StringType, nodeValues)

	// Set API keys if available
	if cluster.APIKeys != nil {
		data.AdminAPIKey = types.StringValue(cluster.APIKeys.Admin)
		data.SearchAPIKey = types.StringValue(cluster.APIKeys.SearchOnly)
	}
}

type clusterPlanWarning struct {
	Attribute string
	Summary   string
	Detail    string
}

func clusterPlanWarnings(state, plan ClusterResourceModel) []clusterPlanWarning {
	var warnings []clusterPlanWarning

	if stringValueChanged(state.SearchDeliveryNetwork, plan.SearchDeliveryNetwork) {
		warnings = append(warnings, clusterPlanWarning{
			Attribute: "search_delivery_network",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud only accepts `search_delivery_network` when a cluster is created. Terraform will replace this cluster to apply the new value.",
		})
	}

	if listValueChanged(state.Regions, plan.Regions) {
		warnings = append(warnings, clusterPlanWarning{
			Attribute: "regions",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud only accepts `regions` when a cluster is created. Terraform will replace this cluster to apply the new region set.",
		})
	}

	if clusterHighAvailabilityReplacementNeeded(state.HighAvailability, plan.HighAvailability) {
		warnings = append(warnings, clusterPlanWarning{
			Attribute: "high_availability",
			Summary:   "Cluster Replacement Required",
			Detail:    "Typesense Cloud does not allow disabling high availability on an existing cluster. Terraform will replace this cluster to apply this change.",
		})
	}

	return warnings
}

func stringValueChanged(state, plan types.String) bool {
	if state.IsNull() || state.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		return false
	}

	return state.ValueString() != plan.ValueString()
}

func listValueChanged(state, plan types.List) bool {
	if state.IsNull() || state.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		return false
	}

	return !state.Equal(plan)
}

func clusterHighAvailabilityReplacementNeeded(state, plan types.String) bool {
	if state.IsNull() || state.IsUnknown() || plan.IsNull() || plan.IsUnknown() {
		return false
	}

	return clusterHighAvailabilityEnabled(state.ValueString()) && !clusterHighAvailabilityEnabled(plan.ValueString())
}

func clusterHighAvailabilityEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "no", "false":
		return false
	default:
		return true
	}
}
