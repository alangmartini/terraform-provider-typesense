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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ConversationModelResource{}
var _ resource.ResourceWithImportState = &ConversationModelResource{}

// NewConversationModelResource creates a new Conversation Model resource
func NewConversationModelResource() resource.Resource {
	return &ConversationModelResource{}
}

// ConversationModelResource defines the resource implementation.
type ConversationModelResource struct {
	client         *client.ServerClient
	featureChecker version.FeatureChecker
}

// ConversationModelResourceModel describes the resource data model.
type ConversationModelResourceModel struct {
	ID                types.String `tfsdk:"id"`
	ModelName         types.String `tfsdk:"model_name"`
	APIKey            types.String `tfsdk:"api_key"`
	HistoryCollection types.String `tfsdk:"history_collection"`
	SystemPrompt      types.String `tfsdk:"system_prompt"`
	TTL               types.Int64  `tfsdk:"ttl"`
	MaxBytes          types.Int64  `tfsdk:"max_bytes"`
	AccountID         types.String `tfsdk:"account_id"`
	VllmURL           types.String `tfsdk:"vllm_url"`
}

func (r *ConversationModelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_conversation_model"
}

func (r *ConversationModelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense Conversation Model (RAG). Conversation Models enable conversational search " +
			"with Retrieval Augmented Generation (RAG), allowing users to ask questions in natural language and " +
			"receive AI-generated answers based on your indexed data.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the conversation model. If not specified, Typesense will auto-generate one.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"model_name": schema.StringAttribute{
				Description: "The LLM model to use for generating responses. Examples: 'openai/gpt-4o', 'openai/gpt-4o-mini', 'cf/meta/llama-3-8b-instruct'.",
				Required:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for authenticating with the LLM provider (OpenAI, Cloudflare, etc.).",
				Required:    true,
				Sensitive:   true,
			},
			"history_collection": schema.StringAttribute{
				Description: "Name of the Typesense collection to store conversation history. This collection must exist before creating the conversation model.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"system_prompt": schema.StringAttribute{
				Description: "Instructions for the LLM that define its behavior, personality, and how it should respond to queries.",
				Required:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "Time-to-live in seconds for conversation history messages. Default is 86400 (24 hours).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(86400),
			},
			"max_bytes": schema.Int64Attribute{
				Description: "Maximum payload size in bytes sent to the LLM per request.",
				Optional:    true,
			},
			"account_id": schema.StringAttribute{
				Description: "Cloudflare account ID. Required when using Cloudflare Workers AI models.",
				Optional:    true,
			},
			"vllm_url": schema.StringAttribute{
				Description: "URL for self-hosted vLLM deployments. Required when using vLLM models.",
				Optional:    true,
			},
		},
	}
}

func (r *ConversationModelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage conversation models.",
		)
		return
	}

	r.client = providerData.ServerClient
	r.featureChecker = providerData.FeatureChecker
}

func (r *ConversationModelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if diags := version.CheckVersionRequirement(r.featureChecker, version.FeatureConversationModels, "typesense_conversation_model"); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var data ConversationModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	model := r.buildConversationModel(&data)

	created, err := r.client.CreateConversationModel(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create conversation model: %s", err))
		return
	}

	// Update model from response (server may return defaults or auto-generated ID)
	r.updateModelFromResponse(&data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ConversationModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	model, err := r.client.GetConversationModel(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read conversation model: %s", err))
		return
	}

	if model == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromResponse(&data, model)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ConversationModelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	model := r.buildConversationModel(&data)

	updated, err := r.client.UpdateConversationModel(ctx, model)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update conversation model: %s", err))
		return
	}

	r.updateModelFromResponse(&data, updated)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ConversationModelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ConversationModelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteConversationModel(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete conversation model: %s", err))
		return
	}
}

func (r *ConversationModelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// buildConversationModel creates a client.ConversationModel from the Terraform resource model
func (r *ConversationModelResource) buildConversationModel(data *ConversationModelResourceModel) *client.ConversationModel {
	model := &client.ConversationModel{
		ModelName:         data.ModelName.ValueString(),
		APIKey:            data.APIKey.ValueString(),
		HistoryCollection: data.HistoryCollection.ValueString(),
		SystemPrompt:      data.SystemPrompt.ValueString(),
	}

	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		model.ID = data.ID.ValueString()
	}

	if !data.TTL.IsNull() {
		model.TTL = data.TTL.ValueInt64()
	}

	if !data.MaxBytes.IsNull() {
		model.MaxBytes = data.MaxBytes.ValueInt64()
	}

	if !data.AccountID.IsNull() {
		model.AccountID = data.AccountID.ValueString()
	}

	if !data.VllmURL.IsNull() {
		model.VllmURL = data.VllmURL.ValueString()
	}

	return model
}

// updateModelFromResponse updates the Terraform resource model from the API response
func (r *ConversationModelResource) updateModelFromResponse(data *ConversationModelResourceModel, model *client.ConversationModel) {
	data.ID = types.StringValue(model.ID)
	data.ModelName = types.StringValue(model.ModelName)
	data.HistoryCollection = types.StringValue(model.HistoryCollection)
	data.SystemPrompt = types.StringValue(model.SystemPrompt)
	// API key is not returned by the API for security, keep the state value

	if model.TTL != 0 {
		data.TTL = types.Int64Value(model.TTL)
	}

	if model.MaxBytes != 0 {
		data.MaxBytes = types.Int64Value(model.MaxBytes)
	}

	if model.AccountID != "" {
		data.AccountID = types.StringValue(model.AccountID)
	}

	if model.VllmURL != "" {
		data.VllmURL = types.StringValue(model.VllmURL)
	}
}
