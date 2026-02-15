package resources

import (
	"context"
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

var _ resource.Resource = &StopwordsSetResource{}
var _ resource.ResourceWithImportState = &StopwordsSetResource{}

// NewStopwordsSetResource creates a new stopwords set resource
func NewStopwordsSetResource() resource.Resource {
	return &StopwordsSetResource{}
}

// StopwordsSetResource defines the resource implementation.
type StopwordsSetResource struct {
	client         *client.ServerClient
	featureChecker version.FeatureChecker
}

// StopwordsSetResourceModel describes the resource data model.
type StopwordsSetResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Stopwords types.Set    `tfsdk:"stopwords"`
	Locale    types.String `tfsdk:"locale"`
}

func (r *StopwordsSetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stopwords_set"
}

func (r *StopwordsSetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense stopwords set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the stopwords set.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name/ID of the stopwords set.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"stopwords": schema.SetAttribute{
				Description: "Set of stopwords.",
				Required:    true,
				ElementType: types.StringType,
			},
			"locale": schema.StringAttribute{
				Description: "Locale for the stopwords (e.g., 'en', 'de').",
				Optional:    true,
			},
		},
	}
}

func (r *StopwordsSetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage stopwords.",
		)
		return
	}

	r.client = providerData.ServerClient
	r.featureChecker = providerData.FeatureChecker
}

func (r *StopwordsSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if diags := version.CheckVersionRequirement(r.featureChecker, version.FeatureStopwords, "typesense_stopwords_set"); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var data StopwordsSetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var stopwords []string
	resp.Diagnostics.Append(data.Stopwords.ElementsAs(ctx, &stopwords, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stopwordsSet := &client.StopwordsSet{
		ID:        data.Name.ValueString(),
		Stopwords: stopwords,
	}

	if !data.Locale.IsNull() {
		stopwordsSet.Locale = data.Locale.ValueString()
	}

	created, err := r.client.CreateStopwordsSet(ctx, stopwordsSet)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create stopwords set: %s", err))
		return
	}

	data.ID = types.StringValue(created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordsSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StopwordsSetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	stopwordsSet, err := r.client.GetStopwordsSet(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read stopwords set: %s", err))
		return
	}

	if stopwordsSet == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update stopwords set
	stopwordValues := make([]types.String, len(stopwordsSet.Stopwords))
	for i, s := range stopwordsSet.Stopwords {
		stopwordValues[i] = types.StringValue(s)
	}
	data.Stopwords, _ = types.SetValueFrom(ctx, types.StringType, stopwordValues)

	if stopwordsSet.Locale != "" {
		data.Locale = types.StringValue(stopwordsSet.Locale)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordsSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StopwordsSetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var stopwords []string
	resp.Diagnostics.Append(data.Stopwords.ElementsAs(ctx, &stopwords, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stopwordsSet := &client.StopwordsSet{
		ID:        data.Name.ValueString(),
		Stopwords: stopwords,
	}

	if !data.Locale.IsNull() {
		stopwordsSet.Locale = data.Locale.ValueString()
	}

	_, err := r.client.CreateStopwordsSet(ctx, stopwordsSet)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update stopwords set: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StopwordsSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StopwordsSetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStopwordsSet(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete stopwords set: %s", err))
		return
	}
}

func (r *StopwordsSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
