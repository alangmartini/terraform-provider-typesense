package resources

import (
	"context"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/alanm/terraform-provider-typesense/internal/version"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &NLSearchModelResource{}
var _ resource.ResourceWithImportState = &NLSearchModelResource{}

// NewNLSearchModelResource creates a new NL search model resource
func NewNLSearchModelResource() resource.Resource {
	return &NLSearchModelResource{}
}

// NLSearchModelResource defines the resource implementation.
type NLSearchModelResource struct {
	client         *client.ServerClient
	featureChecker version.FeatureChecker
}

// NLSearchModelResourceModel describes the resource data model.
type NLSearchModelResourceModel struct {
	ID            types.String  `tfsdk:"id"`
	ModelName     types.String  `tfsdk:"model_name"`
	APIKey        types.String  `tfsdk:"api_key"`
	SystemPrompt  types.String  `tfsdk:"system_prompt"`
	MaxBytes      types.Int64   `tfsdk:"max_bytes"`
	Temperature   types.Float64 `tfsdk:"temperature"`
	TopP          types.Float64 `tfsdk:"top_p"`
	TopK          types.Int64   `tfsdk:"top_k"`
	AccountID     types.String  `tfsdk:"account_id"`
	APIURL        types.String  `tfsdk:"api_url"`
	ProjectID     types.String  `tfsdk:"project_id"`
	AccessToken   types.String  `tfsdk:"access_token"`
	RefreshToken  types.String  `tfsdk:"refresh_token"`
	ClientID      types.String  `tfsdk:"client_id"`
	ClientSecret  types.String  `tfsdk:"client_secret"`
	Region        types.String  `tfsdk:"region"`
	StopSequences types.List    `tfsdk:"stop_sequences"`
	APIVersion    types.String  `tfsdk:"api_version"`
}

func (r *NLSearchModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nl_search_model"
}

func (r *NLSearchModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense Natural Language Search Model. NL Search Models use LLMs to convert " +
			"natural language queries into structured search filters. For example, 'red shoes under $50' can be " +
			"automatically converted to 'filter_by: color:=red && price:<50'.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the NL search model. This ID is used to reference the model in search queries via the nl_model_id parameter.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model_name": schema.StringAttribute{
				Description: "The LLM model to use. Examples: 'openai/gpt-4.1', 'openai/gpt-4o-mini', 'google/gemini-2.5-flash', 'cf/meta/llama-3-8b-instruct'.",
				Required:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for authenticating with the LLM provider (OpenAI, Google, etc.).",
				Required:    true,
				Sensitive:   true,
			},
			"system_prompt": schema.StringAttribute{
				Description: "Custom instructions appended to the Typesense-generated prompt. Use this to provide domain-specific context.",
				Optional:    true,
			},
			"max_bytes": schema.Int64Attribute{
				Description: "Maximum payload size in bytes sent to the LLM.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(16000),
			},
			"temperature": schema.Float64Attribute{
				Description: "Controls randomness in the LLM response (0.0-2.0). Lower values make output more deterministic.",
				Optional:    true,
				Computed:    true,
				Default:     float64default.StaticFloat64(0.0),
			},
			"top_p": schema.Float64Attribute{
				Description: "Nucleus sampling parameter (0.0-1.0). Used primarily with Google models.",
				Optional:    true,
			},
			"top_k": schema.Int64Attribute{
				Description: "Top-k sampling parameter. Limits the number of tokens considered for each step.",
				Optional:    true,
			},
			"account_id": schema.StringAttribute{
				Description: "Cloudflare account ID. Required when using Cloudflare Workers AI models.",
				Optional:    true,
			},
			"api_url": schema.StringAttribute{
				Description: "Custom API URL for self-hosted vLLM models.",
				Optional:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "GCP project ID. Required for Google Vertex AI models.",
				Optional:    true,
			},
			"access_token": schema.StringAttribute{
				Description: "GCP access token. Required for Google Vertex AI models.",
				Optional:    true,
				Sensitive:   true,
			},
			"refresh_token": schema.StringAttribute{
				Description: "GCP refresh token. Required for Google Vertex AI models.",
				Optional:    true,
				Sensitive:   true,
			},
			"client_id": schema.StringAttribute{
				Description: "GCP client ID. Required for Google Vertex AI models.",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				Description: "GCP client secret. Required for Google Vertex AI models.",
				Optional:    true,
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Description: "GCP region for Vertex AI models.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("us-central1"),
			},
			"stop_sequences": schema.ListAttribute{
				Description: "Stop sequences for generation. Used primarily with Google models.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"api_version": schema.StringAttribute{
				Description: "API version for Google models.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("v1beta"),
			},
		},
	}
}

func (r *NLSearchModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage NL search models.",
		)
		return
	}

	r.client = providerData.ServerClient
	r.featureChecker = providerData.FeatureChecker
}

func (r *NLSearchModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if diags := version.CheckVersionRequirement(r.featureChecker, version.FeatureNLSearchModels, "typesense_nl_search_model"); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var data NLSearchModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var diags diag.Diagnostics
	model := r.buildNLSearchModel(ctx, &data, &diags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateNLSearchModel(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create NL search model: %s", err))
		return
	}

	// Update model from response (server may return defaults)
	r.updateModelFromResponse(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NLSearchModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	model, err := r.client.GetNLSearchModel(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read NL search model: %s", err))
		return
	}

	if model == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromResponse(&data, model)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data NLSearchModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var diags diag.Diagnostics
	model := r.buildNLSearchModel(ctx, &data, &diags)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateNLSearchModel(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update NL search model: %s", err))
		return
	}

	r.updateModelFromResponse(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *NLSearchModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NLSearchModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNLSearchModel(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete NL search model: %s", err))
		return
	}
}

func (r *NLSearchModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// buildNLSearchModel creates a client.NLSearchModel from the Terraform resource model
func (r *NLSearchModelResource) buildNLSearchModel(ctx context.Context, data *NLSearchModelResourceModel, diags *diag.Diagnostics) *client.NLSearchModel {
	model := &client.NLSearchModel{
		ID:        data.ID.ValueString(),
		ModelName: data.ModelName.ValueString(),
		APIKey:    data.APIKey.ValueString(),
	}

	if !data.SystemPrompt.IsNull() {
		model.SystemPrompt = data.SystemPrompt.ValueString()
	}

	if !data.MaxBytes.IsNull() {
		model.MaxBytes = data.MaxBytes.ValueInt64()
	}

	if !data.Temperature.IsNull() {
		temp := data.Temperature.ValueFloat64()
		model.Temperature = &temp
	}

	if !data.TopP.IsNull() {
		topP := data.TopP.ValueFloat64()
		model.TopP = &topP
	}

	if !data.TopK.IsNull() {
		topK := data.TopK.ValueInt64()
		model.TopK = &topK
	}

	if !data.AccountID.IsNull() {
		model.AccountID = data.AccountID.ValueString()
	}

	if !data.APIURL.IsNull() {
		model.APIURL = data.APIURL.ValueString()
	}

	if !data.ProjectID.IsNull() {
		model.ProjectID = data.ProjectID.ValueString()
	}

	if !data.AccessToken.IsNull() {
		model.AccessToken = data.AccessToken.ValueString()
	}

	if !data.RefreshToken.IsNull() {
		model.RefreshToken = data.RefreshToken.ValueString()
	}

	if !data.ClientID.IsNull() {
		model.ClientID = data.ClientID.ValueString()
	}

	if !data.ClientSecret.IsNull() {
		model.ClientSecret = data.ClientSecret.ValueString()
	}

	if !data.Region.IsNull() {
		model.Region = data.Region.ValueString()
	}

	if !data.StopSequences.IsNull() {
		var stopSeqs []string
		diags.Append(data.StopSequences.ElementsAs(ctx, &stopSeqs, false)...)
		model.StopSequences = stopSeqs
	}

	if !data.APIVersion.IsNull() {
		model.APIVersion = data.APIVersion.ValueString()
	}

	return model
}

// updateModelFromResponse updates the Terraform resource model from the API response
func (r *NLSearchModelResource) updateModelFromResponse(data *NLSearchModelResourceModel, model *client.NLSearchModel) {
	data.ID = types.StringValue(model.ID)
	data.ModelName = types.StringValue(model.ModelName)
	// API key is not returned by the API for security, keep the state value

	if model.SystemPrompt != "" {
		data.SystemPrompt = types.StringValue(model.SystemPrompt)
	}

	if model.MaxBytes != 0 {
		data.MaxBytes = types.Int64Value(model.MaxBytes)
	}

	if model.Temperature != nil {
		data.Temperature = types.Float64Value(*model.Temperature)
	}

	if model.TopP != nil {
		data.TopP = types.Float64Value(*model.TopP)
	}

	if model.TopK != nil {
		data.TopK = types.Int64Value(*model.TopK)
	}

	if model.AccountID != "" {
		data.AccountID = types.StringValue(model.AccountID)
	}

	if model.APIURL != "" {
		data.APIURL = types.StringValue(model.APIURL)
	}

	if model.ProjectID != "" {
		data.ProjectID = types.StringValue(model.ProjectID)
	}

	if model.Region != "" {
		data.Region = types.StringValue(model.Region)
	}

	if model.APIVersion != "" {
		data.APIVersion = types.StringValue(model.APIVersion)
	}
}
