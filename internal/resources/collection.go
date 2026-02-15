package resources

import (
	"context"
	"encoding/json"
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
)

var _ resource.Resource = &CollectionResource{}
var _ resource.ResourceWithImportState = &CollectionResource{}

// NewCollectionResource creates a new collection resource
func NewCollectionResource() resource.Resource {
	return &CollectionResource{}
}

// CollectionResource defines the resource implementation.
type CollectionResource struct {
	client *client.ServerClient
}

// CollectionResourceModel describes the resource data model.
type CollectionResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Fields              types.List   `tfsdk:"field"`
	DefaultSortingField types.String `tfsdk:"default_sorting_field"`
	TokenSeparators     types.List   `tfsdk:"token_separators"`
	SymbolsToIndex      types.List   `tfsdk:"symbols_to_index"`
	EnableNestedFields  types.Bool   `tfsdk:"enable_nested_fields"`
	NumDocuments        types.Int64  `tfsdk:"num_documents"`
	CreatedAt           types.Int64  `tfsdk:"created_at"`
	Metadata            types.String `tfsdk:"metadata"`
	VoiceQueryModel     types.String `tfsdk:"voice_query_model"`
}

// CollectionFieldModel describes a field in the collection schema
type CollectionFieldModel struct {
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	Facet           types.Bool   `tfsdk:"facet"`
	Optional        types.Bool   `tfsdk:"optional"`
	Index           types.Bool   `tfsdk:"index"`
	Sort            types.Bool   `tfsdk:"sort"`
	Infix           types.Bool   `tfsdk:"infix"`
	Locale          types.String `tfsdk:"locale"`
	NumDim          types.Int64  `tfsdk:"num_dim"`
	VecDist         types.String `tfsdk:"vec_dist"`
	Embed           types.Object `tfsdk:"embed"`
	HnswParams      types.Object `tfsdk:"hnsw_params"`
	Reference       types.String `tfsdk:"reference"`
	AsyncReference  types.Bool   `tfsdk:"async_reference"`
	Stem            types.Bool   `tfsdk:"stem"`
	RangeIndex      types.Bool   `tfsdk:"range_index"`
	Store           types.Bool   `tfsdk:"store"`
	TokenSeparators types.List   `tfsdk:"token_separators"`
	SymbolsToIndex  types.List   `tfsdk:"symbols_to_index"`
}

// embedModelConfigAttrTypes defines the attribute types for the model_config nested object
var embedModelConfigAttrTypes = map[string]attr.Type{
	"model_name": types.StringType,
	"api_key":    types.StringType,
	"url":        types.StringType,
}

// embedAttrTypes defines the attribute types for the embed nested object
var embedAttrTypes = map[string]attr.Type{
	"from":         types.ListType{ElemType: types.StringType},
	"model_config": types.ObjectType{AttrTypes: embedModelConfigAttrTypes},
}

// hnswParamsAttrTypes defines the attribute types for the hnsw_params nested object
var hnswParamsAttrTypes = map[string]attr.Type{
	"ef_construction": types.Int64Type,
	"m":               types.Int64Type,
}

// fieldAttrTypes returns the full attribute type map for a field object
func fieldAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":             types.StringType,
		"type":             types.StringType,
		"facet":            types.BoolType,
		"optional":         types.BoolType,
		"index":            types.BoolType,
		"sort":             types.BoolType,
		"infix":            types.BoolType,
		"locale":           types.StringType,
		"num_dim":          types.Int64Type,
		"vec_dist":         types.StringType,
		"embed":            types.ObjectType{AttrTypes: embedAttrTypes},
		"hnsw_params":      types.ObjectType{AttrTypes: hnswParamsAttrTypes},
		"reference":        types.StringType,
		"async_reference":  types.BoolType,
		"stem":             types.BoolType,
		"range_index":      types.BoolType,
		"store":            types.BoolType,
		"token_separators": types.ListType{ElemType: types.StringType},
		"symbols_to_index": types.ListType{ElemType: types.StringType},
	}
}

func (r *CollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *CollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense collection.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the collection (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the collection.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_sorting_field": schema.StringAttribute{
				Description: "The default field to sort results by.",
				Optional:    true,
			},
			"token_separators": schema.ListAttribute{
				Description: "List of characters to use as token separators.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"symbols_to_index": schema.ListAttribute{
				Description: "List of symbols to index.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"enable_nested_fields": schema.BoolAttribute{
				Description: "Enable nested fields support.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"num_documents": schema.Int64Attribute{
				Description: "Number of documents in the collection.",
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "Timestamp when the collection was created.",
				Computed:    true,
			},
			"metadata": schema.StringAttribute{
				Description: "Custom JSON metadata for the collection. Must be a valid JSON string.",
				Optional:    true,
			},
			"voice_query_model": schema.StringAttribute{
				Description: "Model for voice search (e.g., \"ts/whisper/base.en\").",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"field": schema.ListNestedBlock{
				Description: "Schema fields for the collection.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the field.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "The data type of the field (string, string[], int32, int64, float, bool, geopoint, geopoint[], object, object[], auto, string*, float[]).",
							Required:    true,
						},
						"facet": schema.BoolAttribute{
							Description: "Enable faceting on this field.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"optional": schema.BoolAttribute{
							Description: "Whether the field is optional.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"index": schema.BoolAttribute{
							Description: "Whether to index this field.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"sort": schema.BoolAttribute{
							Description: "Enable sorting on this field. Typesense enables sorting by default for numeric fields (int32, int64, float).",
							Optional:    true,
							Computed:    true,
						},
						"infix": schema.BoolAttribute{
							Description: "Enable infix search on this field.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"locale": schema.StringAttribute{
							Description: "Locale for language-specific processing.",
							Optional:    true,
						},
						"num_dim": schema.Int64Attribute{
							Description: "Number of vector dimensions. When set, a float[] field becomes a vector field.",
							Optional:    true,
						},
						"vec_dist": schema.StringAttribute{
							Description: "Vector distance metric: \"cosine\" or \"ip\". Default: \"cosine\".",
							Optional:    true,
							Computed:    true,
						},
						"embed": schema.SingleNestedAttribute{
							Description: "Auto-embedding configuration for this field.",
							Optional:    true,
							Attributes: map[string]schema.Attribute{
								"from": schema.ListAttribute{
									Description: "List of source field names to generate embeddings from.",
									Required:    true,
									ElementType: types.StringType,
								},
								"model_config": schema.SingleNestedAttribute{
									Description: "Model configuration for auto-embedding.",
									Required:    true,
									Attributes: map[string]schema.Attribute{
										"model_name": schema.StringAttribute{
											Description: "The embedding model name (e.g., \"openai/text-embedding-3-small\").",
											Required:    true,
										},
										"api_key": schema.StringAttribute{
											Description: "API key for the embedding model provider.",
											Optional:    true,
											Sensitive:   true,
										},
										"url": schema.StringAttribute{
											Description: "Custom endpoint URL for the embedding model.",
											Optional:    true,
										},
									},
								},
							},
						},
						"hnsw_params": schema.SingleNestedAttribute{
							Description: "HNSW algorithm tuning parameters for vector fields.",
							Optional:    true,
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"ef_construction": schema.Int64Attribute{
									Description: "HNSW ef_construction parameter. Default: 200.",
									Optional:    true,
									Computed:    true,
								},
								"m": schema.Int64Attribute{
									Description: "HNSW M parameter. Default: 16.",
									Optional:    true,
									Computed:    true,
								},
							},
						},
						"reference": schema.StringAttribute{
							Description: "Reference to another collection field for JOINs (e.g., \"authors.id\"). Cannot be added via update; requires collection recreation.",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						"async_reference": schema.BoolAttribute{
							Description: "Enable async reference for JOINs with large reference sets. Cannot be added via update; requires collection recreation.",
							Optional:    true,
							Computed:    true,
						},
						"stem": schema.BoolAttribute{
							Description: "Enable stemming during indexing for this field.",
							Optional:    true,
							Computed:    true,
						},
						"range_index": schema.BoolAttribute{
							Description: "Optimize this numeric field for range queries.",
							Optional:    true,
							Computed:    true,
						},
						"store": schema.BoolAttribute{
							Description: "Whether to persist this field's data to disk. Default: true.",
							Optional:    true,
							Computed:    true,
						},
						"token_separators": schema.ListAttribute{
							Description: "Field-level token splitting characters.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"symbols_to_index": schema.ListAttribute{
							Description: "Field-level special characters to index.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *CollectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage collections.",
		)
		return
	}

	r.client = providerData.ServerClient
}

func (r *CollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	collection, diags := r.modelToCollection(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateCollection(ctx, collection)
	if err != nil {
		// Check if the collection already exists (HTTP 409 Conflict)
		// If so, adopt the existing collection into state instead of failing
		if strings.Contains(err.Error(), "status 409") {
			existing, getErr := r.client.GetCollection(ctx, data.Name.ValueString())
			if getErr != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Collection already exists but failed to read it: %s", getErr))
				return
			}
			if existing == nil {
				resp.Diagnostics.AddError("Client Error", "Collection reported as existing but could not be found")
				return
			}
			// Adopt the existing collection into state
			r.updateModelFromCollection(ctx, &data, existing)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create collection: %s", err))
		return
	}

	r.updateModelFromCollection(ctx, &data, created)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	collection, err := r.client.GetCollection(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read collection: %s", err))
		return
	}

	if collection == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.updateModelFromCollection(ctx, &data, collection)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CollectionResourceModel
	var state CollectionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get planned and current fields
	plannedFields, diags := r.extractFields(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	currentFields, diags := r.extractFields(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Calculate fields to add and drop
	var fieldsToUpdate []client.CollectionField

	// Find fields to add (in planned but not in current)
	currentFieldNames := make(map[string]bool)
	for _, f := range currentFields {
		currentFieldNames[f.Name] = true
	}

	for _, f := range plannedFields {
		if !currentFieldNames[f.Name] {
			fieldsToUpdate = append(fieldsToUpdate, f)
		}
	}

	// Find fields to drop (in current but not in planned)
	plannedFieldNames := make(map[string]bool)
	for _, f := range plannedFields {
		plannedFieldNames[f.Name] = true
	}

	for _, f := range currentFields {
		if !plannedFieldNames[f.Name] {
			fieldsToUpdate = append(fieldsToUpdate, client.CollectionField{
				Name: f.Name,
				Drop: true,
			})
		}
	}

	// Build the update request
	update := &client.Collection{
		Fields: fieldsToUpdate,
	}

	// Handle collection-level metadata changes
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadata map[string]any
		if err := json.Unmarshal([]byte(data.Metadata.ValueString()), &metadata); err == nil {
			update.Metadata = metadata
		}
	}

	if len(fieldsToUpdate) > 0 || update.Metadata != nil {
		_, err := r.client.UpdateCollection(ctx, data.Name.ValueString(), update)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update collection: %s", err))
			return
		}
	}

	// Re-read the collection to get the updated state
	collection, err := r.client.GetCollection(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read collection after update: %s", err))
		return
	}

	r.updateModelFromCollection(ctx, &data, collection)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CollectionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteCollection(ctx, data.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete collection: %s", err))
		return
	}
}

func (r *CollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func (r *CollectionResource) modelToCollection(ctx context.Context, data *CollectionResourceModel) (*client.Collection, diag.Diagnostics) {
	var diags diag.Diagnostics

	collection := &client.Collection{
		Name:               data.Name.ValueString(),
		EnableNestedFields: data.EnableNestedFields.ValueBool(),
	}

	if !data.DefaultSortingField.IsNull() {
		collection.DefaultSortingField = data.DefaultSortingField.ValueString()
	}

	// Extract token separators
	if !data.TokenSeparators.IsNull() {
		var separators []string
		diags.Append(data.TokenSeparators.ElementsAs(ctx, &separators, false)...)
		collection.TokenSeparators = separators
	}

	// Extract symbols to index
	if !data.SymbolsToIndex.IsNull() {
		var symbols []string
		diags.Append(data.SymbolsToIndex.ElementsAs(ctx, &symbols, false)...)
		collection.SymbolsToIndex = symbols
	}

	// Extract metadata JSON
	if !data.Metadata.IsNull() && !data.Metadata.IsUnknown() {
		var metadata map[string]any
		if err := json.Unmarshal([]byte(data.Metadata.ValueString()), &metadata); err != nil {
			diags.AddError("Invalid Metadata", fmt.Sprintf("The metadata attribute must be a valid JSON string: %s", err))
		} else {
			collection.Metadata = metadata
		}
	}

	// Extract voice query model
	if !data.VoiceQueryModel.IsNull() && !data.VoiceQueryModel.IsUnknown() {
		collection.VoiceQueryModel = data.VoiceQueryModel.ValueString()
	}

	// Extract fields
	fields, d := r.extractFields(ctx, data)
	diags.Append(d...)
	collection.Fields = fields

	return collection, diags
}

func (r *CollectionResource) extractFields(ctx context.Context, data *CollectionResourceModel) ([]client.CollectionField, diag.Diagnostics) {
	var diags diag.Diagnostics
	var fields []client.CollectionField

	if data.Fields.IsNull() || data.Fields.IsUnknown() {
		return fields, diags
	}

	var fieldModels []CollectionFieldModel
	diags.Append(data.Fields.ElementsAs(ctx, &fieldModels, false)...)

	for _, fm := range fieldModels {
		field := client.CollectionField{
			Name:     fm.Name.ValueString(),
			Type:     fm.Type.ValueString(),
			Facet:    fm.Facet.ValueBool(),
			Optional: fm.Optional.ValueBool(),
			Infix:    fm.Infix.ValueBool(),
		}

		if !fm.Index.IsNull() {
			index := fm.Index.ValueBool()
			field.Index = &index
		}

		// Only set Sort if explicitly configured (not null or unknown)
		// This allows Typesense to apply its server-side defaults for numeric types
		if !fm.Sort.IsNull() && !fm.Sort.IsUnknown() {
			sort := fm.Sort.ValueBool()
			field.Sort = &sort
		}

		if !fm.Locale.IsNull() {
			field.Locale = fm.Locale.ValueString()
		}

		// Vector search attributes
		if !fm.NumDim.IsNull() && !fm.NumDim.IsUnknown() {
			field.NumDim = fm.NumDim.ValueInt64()
		}

		if !fm.VecDist.IsNull() && !fm.VecDist.IsUnknown() {
			field.VecDist = fm.VecDist.ValueString()
		}

		// Embed configuration
		if !fm.Embed.IsNull() && !fm.Embed.IsUnknown() {
			embedAttrs := fm.Embed.Attributes()

			var fromFields []string
			if fromVal, ok := embedAttrs["from"]; ok && !fromVal.IsNull() && !fromVal.IsUnknown() {
				fromList := fromVal.(types.List)
				diags.Append(fromList.ElementsAs(ctx, &fromFields, false)...)
			}

			embed := &client.FieldEmbed{
				From: fromFields,
			}

			if mcVal, ok := embedAttrs["model_config"]; ok && !mcVal.IsNull() && !mcVal.IsUnknown() {
				mcAttrs := mcVal.(types.Object).Attributes()

				if mn, ok := mcAttrs["model_name"]; ok && !mn.IsNull() {
					embed.ModelConfig.ModelName = mn.(types.String).ValueString()
				}
				if ak, ok := mcAttrs["api_key"]; ok && !ak.IsNull() && !ak.IsUnknown() {
					embed.ModelConfig.APIKey = ak.(types.String).ValueString()
				}
				if u, ok := mcAttrs["url"]; ok && !u.IsNull() && !u.IsUnknown() {
					embed.ModelConfig.URL = u.(types.String).ValueString()
				}
			}

			field.Embed = embed
		}

		// HNSW params
		if !fm.HnswParams.IsNull() && !fm.HnswParams.IsUnknown() {
			hpAttrs := fm.HnswParams.Attributes()
			hp := &client.FieldHnswParams{}

			if ef, ok := hpAttrs["ef_construction"]; ok && !ef.IsNull() && !ef.IsUnknown() {
				hp.EfConstruction = ef.(types.Int64).ValueInt64()
			}
			if m, ok := hpAttrs["m"]; ok && !m.IsNull() && !m.IsUnknown() {
				hp.M = m.(types.Int64).ValueInt64()
			}

			field.HnswParams = hp
		}

		// Reference / JOINs
		if !fm.Reference.IsNull() && !fm.Reference.IsUnknown() {
			field.Reference = fm.Reference.ValueString()
		}
		if !fm.AsyncReference.IsNull() && !fm.AsyncReference.IsUnknown() {
			v := fm.AsyncReference.ValueBool()
			field.AsyncReference = &v
		}

		// Stem
		if !fm.Stem.IsNull() && !fm.Stem.IsUnknown() {
			stem := fm.Stem.ValueBool()
			field.Stem = &stem
		}

		// Range index
		if !fm.RangeIndex.IsNull() && !fm.RangeIndex.IsUnknown() {
			ri := fm.RangeIndex.ValueBool()
			field.RangeIndex = &ri
		}

		// Store
		if !fm.Store.IsNull() && !fm.Store.IsUnknown() {
			store := fm.Store.ValueBool()
			field.Store = &store
		}

		// Field-level token separators
		if !fm.TokenSeparators.IsNull() && !fm.TokenSeparators.IsUnknown() {
			var seps []string
			diags.Append(fm.TokenSeparators.ElementsAs(ctx, &seps, false)...)
			field.TokenSeparators = seps
		}

		// Field-level symbols to index
		if !fm.SymbolsToIndex.IsNull() && !fm.SymbolsToIndex.IsUnknown() {
			var syms []string
			diags.Append(fm.SymbolsToIndex.ElementsAs(ctx, &syms, false)...)
			field.SymbolsToIndex = syms
		}

		fields = append(fields, field)
	}

	return fields, diags
}

func (r *CollectionResource) updateModelFromCollection(ctx context.Context, data *CollectionResourceModel, collection *client.Collection) {
	data.ID = types.StringValue(collection.Name)
	data.Name = types.StringValue(collection.Name)
	// Handle empty string as null for default_sorting_field
	if collection.DefaultSortingField != "" {
		data.DefaultSortingField = types.StringValue(collection.DefaultSortingField)
	} else {
		data.DefaultSortingField = types.StringNull()
	}
	data.EnableNestedFields = types.BoolValue(collection.EnableNestedFields)
	data.NumDocuments = types.Int64Value(collection.NumDocuments)
	data.CreatedAt = types.Int64Value(collection.CreatedAt)

	// Convert collection-level metadata
	if collection.Metadata != nil {
		metadataBytes, err := json.Marshal(collection.Metadata)
		if err == nil {
			data.Metadata = types.StringValue(string(metadataBytes))
		} else {
			data.Metadata = types.StringNull()
		}
	} else if data.Metadata.IsNull() || data.Metadata.IsUnknown() {
		data.Metadata = types.StringNull()
	}

	// Convert voice query model
	if collection.VoiceQueryModel != "" {
		data.VoiceQueryModel = types.StringValue(collection.VoiceQueryModel)
	} else if data.VoiceQueryModel.IsNull() || data.VoiceQueryModel.IsUnknown() {
		data.VoiceQueryModel = types.StringNull()
	}

	// Convert token separators
	if len(collection.TokenSeparators) > 0 {
		separators := make([]types.String, len(collection.TokenSeparators))
		for i, s := range collection.TokenSeparators {
			separators[i] = types.StringValue(s)
		}
		data.TokenSeparators, _ = types.ListValueFrom(ctx, types.StringType, separators)
	}

	// Convert symbols to index
	if len(collection.SymbolsToIndex) > 0 {
		symbols := make([]types.String, len(collection.SymbolsToIndex))
		for i, s := range collection.SymbolsToIndex {
			symbols[i] = types.StringValue(s)
		}
		data.SymbolsToIndex, _ = types.ListValueFrom(ctx, types.StringType, symbols)
	}

	// Convert fields
	fAttrTypes := fieldAttrTypes()

	// Check if the original model had an 'id' field that we need to preserve.
	// Typesense treats 'id' as an implicit field and doesn't return it in the schema.
	var idFieldValue attr.Value
	if !data.Fields.IsNull() && !data.Fields.IsUnknown() {
		var existingFields []CollectionFieldModel
		data.Fields.ElementsAs(ctx, &existingFields, false)
		for _, ef := range existingFields {
			if ef.Name.ValueString() == "id" {
				idFieldValue = r.buildIdFieldObject(ctx, ef, fAttrTypes)
				break
			}
		}
	}

	// Check if API response contains an 'id' field
	apiHasIdField := false
	for _, f := range collection.Fields {
		if f.Name == "id" {
			apiHasIdField = true
			break
		}
	}

	// Build field values, prepending 'id' if it was in original model but not in API response
	fieldValues := make([]attr.Value, 0, len(collection.Fields)+1)
	if idFieldValue != nil && !apiHasIdField {
		fieldValues = append(fieldValues, idFieldValue)
	}

	for _, f := range collection.Fields {
		fieldObj := r.apiFieldToObjectValue(ctx, f, fAttrTypes)
		fieldValues = append(fieldValues, fieldObj)
	}

	fieldObjType := types.ObjectType{AttrTypes: fAttrTypes}
	data.Fields, _ = types.ListValue(fieldObjType, fieldValues)
}

// buildIdFieldObject creates an object value for the implicit 'id' field
func (r *CollectionResource) buildIdFieldObject(ctx context.Context, ef CollectionFieldModel, fAttrTypes map[string]attr.Type) attr.Value {
	localeVal := types.StringNull()
	if !ef.Locale.IsNull() && ef.Locale.ValueString() != "" {
		localeVal = ef.Locale
	}

	// Resolve computed values to their defaults if unknown/null
	facetVal := ef.Facet
	if facetVal.IsNull() || facetVal.IsUnknown() {
		facetVal = types.BoolValue(false)
	}
	optionalVal := ef.Optional
	if optionalVal.IsNull() || optionalVal.IsUnknown() {
		optionalVal = types.BoolValue(false)
	}
	indexVal := ef.Index
	if indexVal.IsNull() || indexVal.IsUnknown() {
		indexVal = types.BoolValue(true)
	}
	sortVal := ef.Sort
	if sortVal.IsNull() || sortVal.IsUnknown() {
		sortVal = types.BoolValue(false)
	}
	infixVal := ef.Infix
	if infixVal.IsNull() || infixVal.IsUnknown() {
		infixVal = types.BoolValue(false)
	}

	// New field defaults
	numDimVal := types.Int64Null()
	if !ef.NumDim.IsNull() && !ef.NumDim.IsUnknown() {
		numDimVal = ef.NumDim
	}
	vecDistVal := types.StringNull()
	if !ef.VecDist.IsNull() && !ef.VecDist.IsUnknown() {
		vecDistVal = ef.VecDist
	}
	embedVal := types.ObjectNull(embedAttrTypes)
	if !ef.Embed.IsNull() && !ef.Embed.IsUnknown() {
		embedVal = ef.Embed
	}
	hnswVal := types.ObjectNull(hnswParamsAttrTypes)
	if !ef.HnswParams.IsNull() && !ef.HnswParams.IsUnknown() {
		hnswVal = ef.HnswParams
	}
	refVal := types.StringNull()
	if !ef.Reference.IsNull() && !ef.Reference.IsUnknown() {
		refVal = ef.Reference
	}
	asyncRefVal := types.BoolNull()
	if !ef.AsyncReference.IsNull() && !ef.AsyncReference.IsUnknown() {
		asyncRefVal = ef.AsyncReference
	}
	stemVal := types.BoolNull()
	if !ef.Stem.IsNull() && !ef.Stem.IsUnknown() {
		stemVal = ef.Stem
	}
	rangeIndexVal := types.BoolNull()
	if !ef.RangeIndex.IsNull() && !ef.RangeIndex.IsUnknown() {
		rangeIndexVal = ef.RangeIndex
	}
	storeVal := types.BoolNull()
	if !ef.Store.IsNull() && !ef.Store.IsUnknown() {
		storeVal = ef.Store
	}
	fieldTokenSeps := types.ListNull(types.StringType)
	if !ef.TokenSeparators.IsNull() && !ef.TokenSeparators.IsUnknown() {
		fieldTokenSeps = ef.TokenSeparators
	}
	fieldSymsToIndex := types.ListNull(types.StringType)
	if !ef.SymbolsToIndex.IsNull() && !ef.SymbolsToIndex.IsUnknown() {
		fieldSymsToIndex = ef.SymbolsToIndex
	}

	idFieldValue, _ := types.ObjectValue(fAttrTypes, map[string]attr.Value{
		"name":             ef.Name,
		"type":             ef.Type,
		"facet":            facetVal,
		"optional":         optionalVal,
		"index":            indexVal,
		"sort":             sortVal,
		"infix":            infixVal,
		"locale":           localeVal,
		"num_dim":          numDimVal,
		"vec_dist":         vecDistVal,
		"embed":            embedVal,
		"hnsw_params":      hnswVal,
		"reference":        refVal,
		"async_reference":  asyncRefVal,
		"stem":             stemVal,
		"range_index":      rangeIndexVal,
		"store":            storeVal,
		"token_separators": fieldTokenSeps,
		"symbols_to_index": fieldSymsToIndex,
	})
	return idFieldValue
}

// apiFieldToObjectValue converts a client.CollectionField to a Terraform object value
func (r *CollectionResource) apiFieldToObjectValue(ctx context.Context, f client.CollectionField, fAttrTypes map[string]attr.Type) attr.Value {
	indexVal := types.BoolValue(true)
	if f.Index != nil {
		indexVal = types.BoolValue(*f.Index)
	}

	// Handle Sort pointer - if nil, use false as the default display value
	sortVal := types.BoolValue(false)
	if f.Sort != nil {
		sortVal = types.BoolValue(*f.Sort)
	}

	localeVal := types.StringNull()
	if f.Locale != "" {
		localeVal = types.StringValue(f.Locale)
	}

	// num_dim
	numDimVal := types.Int64Null()
	if f.NumDim > 0 {
		numDimVal = types.Int64Value(f.NumDim)
	}

	// vec_dist
	vecDistVal := types.StringNull()
	if f.VecDist != "" {
		vecDistVal = types.StringValue(f.VecDist)
	}

	// embed
	embedVal := types.ObjectNull(embedAttrTypes)
	if f.Embed != nil {
		fromVals := make([]attr.Value, len(f.Embed.From))
		for i, s := range f.Embed.From {
			fromVals[i] = types.StringValue(s)
		}
		fromList, _ := types.ListValue(types.StringType, fromVals)

		apiKeyVal := types.StringNull()
		if f.Embed.ModelConfig.APIKey != "" {
			apiKeyVal = types.StringValue(f.Embed.ModelConfig.APIKey)
		}
		urlVal := types.StringNull()
		if f.Embed.ModelConfig.URL != "" {
			urlVal = types.StringValue(f.Embed.ModelConfig.URL)
		}

		mcObj, _ := types.ObjectValue(embedModelConfigAttrTypes, map[string]attr.Value{
			"model_name": types.StringValue(f.Embed.ModelConfig.ModelName),
			"api_key":    apiKeyVal,
			"url":        urlVal,
		})

		embedVal, _ = types.ObjectValue(embedAttrTypes, map[string]attr.Value{
			"from":         fromList,
			"model_config": mcObj,
		})
	}

	// hnsw_params
	hnswVal := types.ObjectNull(hnswParamsAttrTypes)
	if f.HnswParams != nil {
		hnswVal, _ = types.ObjectValue(hnswParamsAttrTypes, map[string]attr.Value{
			"ef_construction": types.Int64Value(f.HnswParams.EfConstruction),
			"m":               types.Int64Value(f.HnswParams.M),
		})
	}

	// reference
	refVal := types.StringNull()
	if f.Reference != "" {
		refVal = types.StringValue(f.Reference)
	}

	// async_reference
	asyncRefVal := types.BoolNull()
	if f.AsyncReference != nil {
		asyncRefVal = types.BoolValue(*f.AsyncReference)
	}

	// stem
	stemVal := types.BoolNull()
	if f.Stem != nil {
		stemVal = types.BoolValue(*f.Stem)
	}

	// range_index
	rangeIndexVal := types.BoolNull()
	if f.RangeIndex != nil {
		rangeIndexVal = types.BoolValue(*f.RangeIndex)
	}

	// store
	storeVal := types.BoolNull()
	if f.Store != nil {
		storeVal = types.BoolValue(*f.Store)
	}

	// field-level token_separators
	fieldTokenSeps := types.ListNull(types.StringType)
	if len(f.TokenSeparators) > 0 {
		sVals := make([]attr.Value, len(f.TokenSeparators))
		for i, s := range f.TokenSeparators {
			sVals[i] = types.StringValue(s)
		}
		fieldTokenSeps, _ = types.ListValue(types.StringType, sVals)
	}

	// field-level symbols_to_index
	fieldSymsToIndex := types.ListNull(types.StringType)
	if len(f.SymbolsToIndex) > 0 {
		sVals := make([]attr.Value, len(f.SymbolsToIndex))
		for i, s := range f.SymbolsToIndex {
			sVals[i] = types.StringValue(s)
		}
		fieldSymsToIndex, _ = types.ListValue(types.StringType, sVals)
	}

	fieldObj, _ := types.ObjectValue(fAttrTypes, map[string]attr.Value{
		"name":             types.StringValue(f.Name),
		"type":             types.StringValue(f.Type),
		"facet":            types.BoolValue(f.Facet),
		"optional":         types.BoolValue(f.Optional),
		"index":            indexVal,
		"sort":             sortVal,
		"infix":            types.BoolValue(f.Infix),
		"locale":           localeVal,
		"num_dim":          numDimVal,
		"vec_dist":         vecDistVal,
		"embed":            embedVal,
		"hnsw_params":      hnswVal,
		"reference":        refVal,
		"async_reference":  asyncRefVal,
		"stem":             stemVal,
		"range_index":      rangeIndexVal,
		"store":            storeVal,
		"token_separators": fieldTokenSeps,
		"symbols_to_index": fieldSymsToIndex,
	})
	return fieldObj
}
