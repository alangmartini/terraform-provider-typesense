package resources

import (
	"context"
	"fmt"
	"strings"

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

var _ resource.Resource = &SynonymResource{}
var _ resource.ResourceWithImportState = &SynonymResource{}

// NewSynonymResource creates a new synonym resource
func NewSynonymResource() resource.Resource {
	return &SynonymResource{}
}

// SynonymResource defines the resource implementation.
type SynonymResource struct {
	client         *client.ServerClient
	featureChecker version.FeatureChecker
}

// SynonymResourceModel describes the resource data model.
type SynonymResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Collection types.String `tfsdk:"collection"`
	Name       types.String `tfsdk:"name"`
	Root       types.String `tfsdk:"root"`
	Synonyms   types.List   `tfsdk:"synonyms"`
}

func (r *SynonymResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_synonym"
}

func (r *SynonymResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Typesense synonym configuration for a collection. In Typesense v29 and earlier, synonyms are per-collection. In v30+, synonyms are managed via synonym sets at the system level (the collection name becomes the synonym set name).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier (collection/name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"collection": schema.StringAttribute{
				Description: "The name of the collection this synonym belongs to. In v30+, this becomes the synonym set name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name/ID of the synonym rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"root": schema.StringAttribute{
				Description: "For one-way synonyms, the root word that the synonyms map to. Leave empty for multi-way synonyms.",
				Optional:    true,
			},
			"synonyms": schema.ListAttribute{
				Description: "List of synonym words.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *SynonymResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
			"The server_host and server_api_key must be configured in the provider to manage synonyms.",
		)
		return
	}

	r.client = providerData.ServerClient
	r.featureChecker = providerData.FeatureChecker
}

func (r *SynonymResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var synonyms []string
	resp.Diagnostics.Append(data.Synonyms.ElementsAs(ctx, &synonyms, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection := data.Collection.ValueString()
	name := data.Name.ValueString()
	root := ""
	if !data.Root.IsNull() {
		root = data.Root.ValueString()
	}

	// Use version-appropriate API
	if r.featureChecker.SupportsFeature(version.FeatureSynonymSets) {
		// v30+: Use synonym sets API
		err := r.createSynonymV30(ctx, collection, name, root, synonyms)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create synonym: %s", err))
			return
		}
	} else {
		// v29 and earlier: Use per-collection synonyms API
		synonym := &client.Synonym{
			ID:       name,
			Synonyms: synonyms,
			Root:     root,
		}

		_, err := r.client.CreateSynonym(ctx, collection, synonym)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create synonym: %s", err))
			return
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", collection, name))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	collection := data.Collection.ValueString()
	name := data.Name.ValueString()

	var synonyms []string
	var root string
	var found bool

	// Use version-appropriate API
	if r.featureChecker.SupportsFeature(version.FeatureSynonymSets) {
		// v30+: Use synonym sets API
		synItem, err := r.getSynonymV30(ctx, collection, name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read synonym: %s", err))
			return
		}
		if synItem != nil {
			found = true
			synonyms = synItem.Synonyms
			root = synItem.Root
		}
	} else {
		// v29 and earlier: Use per-collection synonyms API
		synonym, err := r.client.GetSynonym(ctx, collection, name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read synonym: %s", err))
			return
		}
		if synonym != nil {
			found = true
			synonyms = synonym.Synonyms
			root = synonym.Root
		}
	}

	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update synonyms list
	synonymValues := make([]types.String, len(synonyms))
	for i, s := range synonyms {
		synonymValues[i] = types.StringValue(s)
	}
	data.Synonyms, _ = types.ListValueFrom(ctx, types.StringType, synonymValues)

	if root != "" {
		data.Root = types.StringValue(root)
	} else {
		data.Root = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var synonyms []string
	resp.Diagnostics.Append(data.Synonyms.ElementsAs(ctx, &synonyms, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection := data.Collection.ValueString()
	name := data.Name.ValueString()
	root := ""
	if !data.Root.IsNull() {
		root = data.Root.ValueString()
	}

	// Use version-appropriate API
	if r.featureChecker.SupportsFeature(version.FeatureSynonymSets) {
		// v30+: Use synonym sets API (same as create - upsert behavior)
		err := r.createSynonymV30(ctx, collection, name, root, synonyms)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update synonym: %s", err))
			return
		}
	} else {
		// v29 and earlier: Use per-collection synonyms API
		synonym := &client.Synonym{
			ID:       name,
			Synonyms: synonyms,
			Root:     root,
		}

		_, err := r.client.CreateSynonym(ctx, collection, synonym)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update synonym: %s", err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SynonymResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SynonymResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	collection := data.Collection.ValueString()
	name := data.Name.ValueString()

	// Use version-appropriate API
	if r.featureChecker.SupportsFeature(version.FeatureSynonymSets) {
		// v30+: Use synonym sets API
		err := r.deleteSynonymV30(ctx, collection, name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete synonym: %s", err))
			return
		}
	} else {
		// v29 and earlier: Use per-collection synonyms API
		err := r.client.DeleteSynonym(ctx, collection, name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete synonym: %s", err))
			return
		}
	}
}

func (r *SynonymResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: collection/synonym_name
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: collection/synonym_name, got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("collection"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}

// v30+ helper methods for synonym sets

// createSynonymV30 creates or updates a synonym using the v30 synonym sets API.
// The collection name is used as the synonym set name.
func (r *SynonymResource) createSynonymV30(ctx context.Context, collection, name, root string, synonyms []string) error {
	// Get existing synonym set or create new one
	existingSet, err := r.client.GetSynonymSet(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to get synonym set: %w", err)
	}

	var synSet *client.SynonymSet
	if existingSet == nil {
		// Create new synonym set
		synSet = &client.SynonymSet{
			Name:     collection,
			Synonyms: []client.SynonymItem{},
		}
	} else {
		synSet = existingSet
	}

	// Find and update or add the synonym item
	found := false
	for i, item := range synSet.Synonyms {
		if item.ID == name {
			synSet.Synonyms[i] = client.SynonymItem{
				ID:       name,
				Root:     root,
				Synonyms: synonyms,
			}
			found = true
			break
		}
	}

	if !found {
		synSet.Synonyms = append(synSet.Synonyms, client.SynonymItem{
			ID:       name,
			Root:     root,
			Synonyms: synonyms,
		})
	}

	// Upsert the synonym set
	_, err = r.client.UpsertSynonymSet(ctx, synSet)
	if err != nil {
		return fmt.Errorf("failed to upsert synonym set: %w", err)
	}

	return nil
}

// getSynonymV30 retrieves a specific synonym from a v30 synonym set.
func (r *SynonymResource) getSynonymV30(ctx context.Context, collection, name string) (*client.SynonymItem, error) {
	synSet, err := r.client.GetSynonymSet(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to get synonym set: %w", err)
	}

	if synSet == nil {
		return nil, nil
	}

	for _, item := range synSet.Synonyms {
		if item.ID == name {
			return &item, nil
		}
	}

	return nil, nil
}

// deleteSynonymV30 removes a synonym from a v30 synonym set.
// If the synonym set becomes empty, it deletes the entire set.
func (r *SynonymResource) deleteSynonymV30(ctx context.Context, collection, name string) error {
	synSet, err := r.client.GetSynonymSet(ctx, collection)
	if err != nil {
		return fmt.Errorf("failed to get synonym set: %w", err)
	}

	if synSet == nil {
		// Already deleted
		return nil
	}

	// Remove the synonym item
	newSynonyms := make([]client.SynonymItem, 0, len(synSet.Synonyms))
	for _, item := range synSet.Synonyms {
		if item.ID != name {
			newSynonyms = append(newSynonyms, item)
		}
	}

	if len(newSynonyms) == 0 {
		// Delete the entire synonym set if empty
		return r.client.DeleteSynonymSet(ctx, collection)
	}

	// Update the synonym set without the deleted item
	synSet.Synonyms = newSynonyms
	_, err = r.client.UpsertSynonymSet(ctx, synSet)
	if err != nil {
		return fmt.Errorf("failed to update synonym set: %w", err)
	}

	return nil
}
