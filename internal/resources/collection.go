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
	Name                types.String `tfsdk:"name"`
	Fields              types.List   `tfsdk:"field"`
	DefaultSortingField types.String `tfsdk:"default_sorting_field"`
	TokenSeparators     types.List   `tfsdk:"token_separators"`
	SymbolsToIndex      types.List   `tfsdk:"symbols_to_index"`
	EnableNestedFields  types.Bool   `tfsdk:"enable_nested_fields"`
	NumDocuments        types.Int64  `tfsdk:"num_documents"`
	CreatedAt           types.Int64  `tfsdk:"created_at"`
}

// CollectionFieldModel describes a field in the collection schema
type CollectionFieldModel struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Facet    types.Bool   `tfsdk:"facet"`
	Optional types.Bool   `tfsdk:"optional"`
	Index    types.Bool   `tfsdk:"index"`
	Sort     types.Bool   `tfsdk:"sort"`
	Infix    types.Bool   `tfsdk:"infix"`
	Locale   types.String `tfsdk:"locale"`
}

func (r *CollectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (r *CollectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense collection.",
		Attributes: map[string]schema.Attribute{
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
							Description: "The data type of the field (string, string[], int32, int64, float, bool, geopoint, geopoint[], object, object[], auto, string*).",
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

	if len(fieldsToUpdate) > 0 {
		update := &client.Collection{
			Fields: fieldsToUpdate,
		}

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
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
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
			Sort:     fm.Sort.ValueBool(),
			Infix:    fm.Infix.ValueBool(),
		}

		if !fm.Index.IsNull() {
			index := fm.Index.ValueBool()
			field.Index = &index
		}

		if !fm.Locale.IsNull() {
			field.Locale = fm.Locale.ValueString()
		}

		fields = append(fields, field)
	}

	return fields, diags
}

func (r *CollectionResource) updateModelFromCollection(ctx context.Context, data *CollectionResourceModel, collection *client.Collection) {
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
	fieldAttrTypes := map[string]attr.Type{
		"name":     types.StringType,
		"type":     types.StringType,
		"facet":    types.BoolType,
		"optional": types.BoolType,
		"index":    types.BoolType,
		"sort":     types.BoolType,
		"infix":    types.BoolType,
		"locale":   types.StringType,
	}

	// Check if the original model had an 'id' field that we need to preserve.
	// Typesense treats 'id' as an implicit field and doesn't return it in the schema.
	var idFieldValue attr.Value
	if !data.Fields.IsNull() && !data.Fields.IsUnknown() {
		var existingFields []CollectionFieldModel
		data.Fields.ElementsAs(ctx, &existingFields, false)
		for _, ef := range existingFields {
			if ef.Name.ValueString() == "id" {
				// Preserve the id field from the original plan/state
				localeVal := types.StringNull()
				if !ef.Locale.IsNull() && ef.Locale.ValueString() != "" {
					localeVal = ef.Locale
				}
				idFieldValue, _ = types.ObjectValue(fieldAttrTypes, map[string]attr.Value{
					"name":     ef.Name,
					"type":     ef.Type,
					"facet":    ef.Facet,
					"optional": ef.Optional,
					"index":    ef.Index,
					"sort":     ef.Sort,
					"infix":    ef.Infix,
					"locale":   localeVal,
				})
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
		indexVal := types.BoolValue(true)
		if f.Index != nil {
			indexVal = types.BoolValue(*f.Index)
		}

		localeVal := types.StringNull()
		if f.Locale != "" {
			localeVal = types.StringValue(f.Locale)
		}

		fieldObj, _ := types.ObjectValue(fieldAttrTypes, map[string]attr.Value{
			"name":     types.StringValue(f.Name),
			"type":     types.StringValue(f.Type),
			"facet":    types.BoolValue(f.Facet),
			"optional": types.BoolValue(f.Optional),
			"index":    indexVal,
			"sort":     types.BoolValue(f.Sort),
			"infix":    types.BoolValue(f.Infix),
			"locale":   localeVal,
		})
		fieldValues = append(fieldValues, fieldObj)
	}

	fieldObjType := types.ObjectType{AttrTypes: fieldAttrTypes}
	data.Fields, _ = types.ListValue(fieldObjType, fieldValues)
}
