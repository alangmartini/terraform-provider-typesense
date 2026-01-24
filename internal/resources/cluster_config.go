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

var _ resource.Resource = &ClusterConfigChangeResource{}
var _ resource.ResourceWithImportState = &ClusterConfigChangeResource{}

// NewClusterConfigChangeResource creates a new cluster config change resource
func NewClusterConfigChangeResource() resource.Resource {
	return &ClusterConfigChangeResource{}
}

// ClusterConfigChangeResource defines the resource implementation.
type ClusterConfigChangeResource struct {
	client *client.CloudClient
}

// ClusterConfigChangeResourceModel describes the resource data model.
type ClusterConfigChangeResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ClusterID           types.String `tfsdk:"cluster_id"`
	NewMemory           types.String `tfsdk:"new_memory"`
	NewVCPU             types.String `tfsdk:"new_vcpu"`
	NewHighAvailability types.String `tfsdk:"new_high_availability"`
	NewTypesenseVersion types.String `tfsdk:"new_typesense_server_version"`
	PerformChangeAt     types.Int64  `tfsdk:"perform_change_at"`
	Status              types.String `tfsdk:"status"`
}

func (r *ClusterConfigChangeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_config_change"
}

func (r *ClusterConfigChangeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Schedules a configuration change for a Typesense Cloud cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the configuration change.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description: "The ID of the cluster to change.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"new_memory": schema.StringAttribute{
				Description: "New memory configuration.",
				Optional:    true,
			},
			"new_vcpu": schema.StringAttribute{
				Description: "New vCPU configuration.",
				Optional:    true,
			},
			"new_high_availability": schema.StringAttribute{
				Description: "New high availability setting.",
				Optional:    true,
			},
			"new_typesense_server_version": schema.StringAttribute{
				Description: "New Typesense server version.",
				Optional:    true,
			},
			"perform_change_at": schema.Int64Attribute{
				Description: "Unix timestamp when to perform the change. If not specified, change is performed immediately.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the configuration change.",
				Computed:    true,
			},
		},
	}
}

func (r *ClusterConfigChangeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	if providerData.CloudClient == nil {
		resp.Diagnostics.AddError(
			"Cloud Management API Not Configured",
			"The cloud_management_api_key must be configured in the provider to manage cluster configuration changes.",
		)
		return
	}

	r.client = providerData.CloudClient
}

func (r *ClusterConfigChangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ClusterConfigChangeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	change := &client.ClusterConfigChange{
		ClusterID: data.ClusterID.ValueString(),
	}

	if !data.NewMemory.IsNull() {
		change.NewMemory = data.NewMemory.ValueString()
	}
	if !data.NewVCPU.IsNull() {
		change.NewVCPU = data.NewVCPU.ValueString()
	}
	if !data.NewHighAvailability.IsNull() {
		change.NewHighAvailability = data.NewHighAvailability.ValueString()
	}
	if !data.NewTypesenseVersion.IsNull() {
		change.NewTypesenseVersion = data.NewTypesenseVersion.ValueString()
	}
	if !data.PerformChangeAt.IsNull() {
		change.PerformChangeAt = data.PerformChangeAt.ValueInt64()
	}

	created, err := r.client.CreateClusterConfigChange(ctx, change)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create cluster config change: %s", err))
		return
	}

	data.ID = types.StringValue(created.ID)
	data.Status = types.StringValue(created.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterConfigChangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ClusterConfigChangeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	change, err := r.client.GetClusterConfigChange(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read cluster config change: %s", err))
		return
	}

	if change == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.Status = types.StringValue(change.Status)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ClusterConfigChangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Configuration changes cannot be updated after creation
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Cluster configuration changes cannot be updated after creation. Delete and recreate the resource to schedule a new change.",
	)
}

func (r *ClusterConfigChangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ClusterConfigChangeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteClusterConfigChange(ctx, data.ClusterID.ValueString(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete cluster config change: %s", err))
		return
	}
}

func (r *ClusterConfigChangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
