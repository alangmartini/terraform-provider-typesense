package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"

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

// synonymSetMu serializes synonym set creation to prevent race conditions
// where concurrent creates could overwrite each other's items via set-level PUT.
var synonymSetMu sync.Map // map[string]*sync.Mutex

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
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to create synonym using v30+ synonym sets API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
			return
		}
	} else if r.featureChecker.SupportsFeature(version.FeaturePerCollectionSynonyms) || r.featureChecker.GetVersion() == nil {
		// v29 and earlier (or unknown version): Use per-collection synonyms API
		synonym := &client.Synonym{
			ID:       name,
			Synonyms: synonyms,
			Root:     root,
		}

		_, err := r.client.CreateSynonym(ctx, collection, synonym)
		if err != nil {
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to create synonym using per-collection synonyms API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s). Note: Per-collection synonyms were removed in v30+. Use synonym sets in v30+.", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
			return
		}
	} else {
		serverVer := r.featureChecker.GetVersion()
		resp.Diagnostics.AddError(
			"Unsupported Typesense Version for Synonyms",
			fmt.Sprintf(
				"Your Typesense server (v%s) does not support any known synonym API. "+
					"Per-collection synonyms require v29 or earlier, synonym sets require v30+.",
				serverVer.String(),
			),
		)
		return
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
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to read synonym using v30+ synonym sets API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
			return
		}
		if synItem != nil {
			found = true
			synonyms = synItem.Synonyms
			root = synItem.Root
		}
	} else {
		// v29 and earlier (or unknown version): Use per-collection synonyms API
		synonym, err := r.client.GetSynonym(ctx, collection, name)
		if err != nil {
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to read synonym using per-collection synonyms API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
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
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to update synonym using v30+ synonym sets API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
			return
		}
	} else {
		// v29 and earlier (or unknown version): Use per-collection synonyms API
		synonym := &client.Synonym{
			ID:       name,
			Synonyms: synonyms,
			Root:     root,
		}

		_, err := r.client.CreateSynonym(ctx, collection, synonym)
		if err != nil {
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to update synonym using per-collection synonyms API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
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
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to delete synonym using v30+ synonym sets API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
			return
		}
	} else {
		// v29 and earlier (or unknown version): Use per-collection synonyms API
		err := r.client.DeleteSynonym(ctx, collection, name)
		if err != nil {
			serverVer := r.featureChecker.GetVersion()
			detail := fmt.Sprintf("Unable to delete synonym using per-collection synonyms API: %s", err)
			if serverVer != nil {
				detail += fmt.Sprintf(" (server version: v%s)", serverVer.String())
			}
			resp.Diagnostics.AddError("Client Error", detail)
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

// getSetMutex returns a per-collection mutex for serializing synonym set creation.
func getSetMutex(collection string) *sync.Mutex {
	mu, _ := synonymSetMu.LoadOrStore(collection, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

// ensureSynonymSetExists ensures the synonym set for a collection exists, creating it if needed.
// Uses a per-collection mutex to prevent the race condition where concurrent empty-set creates
// could overwrite items added by other goroutines.
func (r *SynonymResource) ensureSynonymSetExists(ctx context.Context, collection string) error {
	mu := getSetMutex(collection)
	mu.Lock()
	defer mu.Unlock()

	return r.client.EnsureSynonymSetExists(ctx, collection)
}

// createSynonymV30 creates or updates a synonym using the v30 synonym sets item-level API.
// The collection name is used as the synonym set name.
func (r *SynonymResource) createSynonymV30(ctx context.Context, collection, name, root string, synonyms []string) error {
	// Ensure the synonym set exists (serialized per collection)
	if err := r.ensureSynonymSetExists(ctx, collection); err != nil {
		return fmt.Errorf("failed to ensure synonym set: %w", err)
	}

	// Use item-level API (safe for concurrent access)
	item := &client.SynonymItem{
		ID:       name,
		Root:     root,
		Synonyms: synonyms,
	}
	_, err := r.client.UpsertSynonymSetItem(ctx, collection, item)
	if err != nil {
		return fmt.Errorf("failed to upsert synonym item: %w", err)
	}

	return nil
}

// getSynonymV30 retrieves a specific synonym from a v30 synonym set.
func (r *SynonymResource) getSynonymV30(ctx context.Context, collection, name string) (*client.SynonymItem, error) {
	return r.client.GetSynonymSetItem(ctx, collection, name)
}

// deleteSynonymV30 removes a synonym from a v30 synonym set.
func (r *SynonymResource) deleteSynonymV30(ctx context.Context, collection, name string) error {
	return r.client.DeleteSynonymSetItem(ctx, collection, name)
}
