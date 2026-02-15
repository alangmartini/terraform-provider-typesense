package resources

import (
	"context"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	providertypes "github.com/alanm/terraform-provider-typesense/internal/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &StemmingDictionaryResource{}
var _ resource.ResourceWithImportState = &StemmingDictionaryResource{}

// NewStemmingDictionaryResource creates a new stemming dictionary resource
func NewStemmingDictionaryResource() resource.Resource {
	return &StemmingDictionaryResource{}
}

// StemmingDictionaryResource defines the resource implementation.
type StemmingDictionaryResource struct {
	client *client.ServerClient
}

// StemmingDictionaryResourceModel describes the resource data model.
type StemmingDictionaryResourceModel struct {
	ID           types.String `tfsdk:"id"`
	DictionaryID types.String `tfsdk:"dictionary_id"`
	Words        types.List   `tfsdk:"words"`
}

// wordStemMappingAttrTypes defines the attribute types for a word-stem mapping object
var wordStemMappingAttrTypes = map[string]attr.Type{
	"word": types.StringType,
	"stem": types.StringType,
}

func (r *StemmingDictionaryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stemming_dictionary"
}

func (r *StemmingDictionaryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense stemming dictionary.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the stemming dictionary.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dictionary_id": schema.StringAttribute{
				Description: "The dictionary identifier.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"words": schema.ListNestedAttribute{
				Description: "List of word-to-stem mappings.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"word": schema.StringAttribute{
							Description: "The word to stem.",
							Required:    true,
						},
						"stem": schema.StringAttribute{
							Description: "The stem to map to.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *StemmingDictionaryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage stemming dictionaries.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *StemmingDictionaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StemmingDictionaryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	words, diags := extractWords(ctx, data.Words)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictID := data.DictionaryID.ValueString()
	_, err := r.client.UpsertStemmingDictionary(ctx, dictID, words)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create stemming dictionary: %s", err))
		return
	}

	data.ID = types.StringValue(dictID)
	// Keep the planned word order (API may return in a different order)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StemmingDictionaryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	dict, err := r.client.GetStemmingDictionary(ctx, data.DictionaryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read stemming dictionary: %s", err))
		return
	}

	if dict == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Reconcile API response with state order to avoid spurious diffs
	// The API may return words in a different order than the user specified
	stateWords, diags := extractWords(ctx, data.Words)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build lookup of API words by word key
	apiWordMap := make(map[string]string, len(dict.Words))
	for _, w := range dict.Words {
		apiWordMap[w.Word] = w.Stem
	}

	// Check if state words match API words (same content, possibly different order)
	stateMatchesAPI := len(stateWords) == len(dict.Words)
	if stateMatchesAPI {
		for _, sw := range stateWords {
			if apiStem, ok := apiWordMap[sw.Word]; !ok || apiStem != sw.Stem {
				stateMatchesAPI = false
				break
			}
		}
	}

	// Only update words if content actually changed (not just reordered)
	if !stateMatchesAPI {
		data.Words = wordsToListValue(ctx, dict.Words)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StemmingDictionaryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	words, diags := extractWords(ctx, data.Words)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dictID := data.DictionaryID.ValueString()
	_, err := r.client.UpsertStemmingDictionary(ctx, dictID, words)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update stemming dictionary: %s", err))
		return
	}

	// Keep the planned word order (API may return in a different order)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StemmingDictionaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StemmingDictionaryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteStemmingDictionary(ctx, data.DictionaryID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete stemming dictionary: %s", err))
		return
	}
}

func (r *StemmingDictionaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("dictionary_id"), req.ID)...)
}

// extractWords converts the Terraform list of word-stem objects to client WordStemMapping slice
func extractWords(ctx context.Context, wordsList types.List) ([]client.WordStemMapping, diag.Diagnostics) {
	var diags diag.Diagnostics

	type wordStemModel struct {
		Word types.String `tfsdk:"word"`
		Stem types.String `tfsdk:"stem"`
	}

	var models []wordStemModel
	diags.Append(wordsList.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}

	words := make([]client.WordStemMapping, len(models))
	for i, m := range models {
		words[i] = client.WordStemMapping{
			Word: m.Word.ValueString(),
			Stem: m.Stem.ValueString(),
		}
	}
	return words, diags
}

// wordsToListValue converts client WordStemMapping slice to a Terraform list value
func wordsToListValue(ctx context.Context, words []client.WordStemMapping) types.List {
	elems := make([]attr.Value, len(words))
	for i, w := range words {
		elems[i], _ = types.ObjectValue(wordStemMappingAttrTypes, map[string]attr.Value{
			"word": types.StringValue(w.Word),
			"stem": types.StringValue(w.Stem),
		})
	}
	list, _ := types.ListValue(types.ObjectType{AttrTypes: wordStemMappingAttrTypes}, elems)
	return list
}
