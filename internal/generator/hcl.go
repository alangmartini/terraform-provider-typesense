package generator

import (
	"encoding/json"
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const clusterSectionMutabilityComment = `# Cluster update behavior:
# - In-place via PATCH: name, auto_upgrade_capacity
# - In-place via configuration changes: memory, vcpu, high_availability, typesense_server_version
# - Requires a new cluster: regions, search_delivery_network

`

// generateTerraformBlock creates the terraform required_providers block
func generateTerraformBlock(f *hclwrite.File) {
	tfBlock := f.Body().AppendNewBlock("terraform", nil)
	reqProviders := tfBlock.Body().AppendNewBlock("required_providers", nil)
	reqProviders.Body().SetAttributeValue("typesense", cty.ObjectVal(map[string]cty.Value{
		"source": cty.StringVal(tfnames.ProviderSource),
	}))
	f.Body().AppendNewline()
}

// generateProviderBlock creates the provider configuration block
func generateProviderBlock(f *hclwrite.File, host string, port int, protocol string, includeServerAPIKey bool, includeCloudAPIKey bool) {
	providerBlock := f.Body().AppendNewBlock("provider", []string{"typesense"})
	providerBlock.Body().SetAttributeValue("server_host", cty.StringVal(host))
	providerBlock.Body().SetAttributeValue("server_port", cty.NumberIntVal(int64(port)))
	providerBlock.Body().SetAttributeValue("server_protocol", cty.StringVal(protocol))
	if includeServerAPIKey {
		providerBlock.Body().AppendUnstructuredTokens(hclwrite.Tokens{
			{Type: 4, Bytes: []byte("# server_api_key = \"YOUR_API_KEY_HERE\"\n")}, // TokenComment = 4
		})
	}
	if includeCloudAPIKey {
		providerBlock.Body().AppendUnstructuredTokens(hclwrite.Tokens{
			{Type: 4, Bytes: []byte("# cloud_management_api_key = \"YOUR_CLOUD_API_KEY_HERE\"\n")}, // TokenComment = 4
		})
	}
	f.Body().AppendNewline()
}

// generateCollectionBlock creates an HCL block for a collection resource
func generateCollectionBlock(c *client.Collection, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceCollection), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(c.Name))

	if c.DefaultSortingField != "" {
		body.SetAttributeValue("default_sorting_field", cty.StringVal(c.DefaultSortingField))
	}

	if c.EnableNestedFields {
		body.SetAttributeValue("enable_nested_fields", cty.BoolVal(true))
	}

	if len(c.TokenSeparators) > 0 {
		vals := make([]cty.Value, len(c.TokenSeparators))
		for i, v := range c.TokenSeparators {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("token_separators", cty.ListVal(vals))
	}

	if len(c.SymbolsToIndex) > 0 {
		vals := make([]cty.Value, len(c.SymbolsToIndex))
		for i, v := range c.SymbolsToIndex {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("symbols_to_index", cty.ListVal(vals))
	}

	// Add fields
	for _, field := range c.Fields {
		fieldBlock := body.AppendNewBlock("field", nil)
		fieldBody := fieldBlock.Body()

		fieldBody.SetAttributeValue("name", cty.StringVal(field.Name))
		fieldBody.SetAttributeValue("type", cty.StringVal(field.Type))

		if field.Facet {
			fieldBody.SetAttributeValue("facet", cty.BoolVal(true))
		}
		if field.Optional {
			fieldBody.SetAttributeValue("optional", cty.BoolVal(true))
		}
		if field.Index != nil && !*field.Index {
			fieldBody.SetAttributeValue("index", cty.BoolVal(false))
		}
		if field.Sort != nil && *field.Sort {
			fieldBody.SetAttributeValue("sort", cty.BoolVal(true))
		}
		if field.Infix {
			fieldBody.SetAttributeValue("infix", cty.BoolVal(true))
		}
		if field.Locale != "" {
			fieldBody.SetAttributeValue("locale", cty.StringVal(field.Locale))
		}
		if field.NumDim > 0 {
			fieldBody.SetAttributeValue("num_dim", cty.NumberIntVal(field.NumDim))
		}
		if field.VecDist != "" {
			fieldBody.SetAttributeValue("vec_dist", cty.StringVal(field.VecDist))
		}
		if field.Reference != "" {
			fieldBody.SetAttributeValue("reference", cty.StringVal(field.Reference))
		}
		if field.AsyncReference != nil && *field.AsyncReference {
			fieldBody.SetAttributeValue("async_reference", cty.BoolVal(true))
		}
		if field.Stem != nil && *field.Stem {
			fieldBody.SetAttributeValue("stem", cty.BoolVal(true))
		}
		if field.RangeIndex != nil && *field.RangeIndex {
			fieldBody.SetAttributeValue("range_index", cty.BoolVal(true))
		}
		if field.Store != nil && !*field.Store {
			fieldBody.SetAttributeValue("store", cty.BoolVal(false))
		}
		if len(field.TokenSeparators) > 0 {
			sVals := make([]cty.Value, len(field.TokenSeparators))
			for i, v := range field.TokenSeparators {
				sVals[i] = cty.StringVal(v)
			}
			fieldBody.SetAttributeValue("token_separators", cty.ListVal(sVals))
		}
		if len(field.SymbolsToIndex) > 0 {
			sVals := make([]cty.Value, len(field.SymbolsToIndex))
			for i, v := range field.SymbolsToIndex {
				sVals[i] = cty.StringVal(v)
			}
			fieldBody.SetAttributeValue("symbols_to_index", cty.ListVal(sVals))
		}
		if field.Embed != nil {
			embedVals := make(map[string]cty.Value)
			if len(field.Embed.From) > 0 {
				fromVals := make([]cty.Value, len(field.Embed.From))
				for i, v := range field.Embed.From {
					fromVals[i] = cty.StringVal(v)
				}
				embedVals["from"] = cty.ListVal(fromVals)
			}
			modelConfigVals := map[string]cty.Value{
				"model_name": cty.StringVal(field.Embed.ModelConfig.ModelName),
			}
			if field.Embed.ModelConfig.URL != "" {
				modelConfigVals["url"] = cty.StringVal(field.Embed.ModelConfig.URL)
			}
			// Intentionally omit api_key from generated HCL (sensitive)
			embedVals["model_config"] = cty.ObjectVal(modelConfigVals)
			fieldBody.SetAttributeValue("embed", cty.ObjectVal(embedVals))
		}
		if field.HnswParams != nil {
			hnswVals := make(map[string]cty.Value)
			if field.HnswParams.EfConstruction > 0 {
				hnswVals["ef_construction"] = cty.NumberIntVal(field.HnswParams.EfConstruction)
			}
			if field.HnswParams.M > 0 {
				hnswVals["m"] = cty.NumberIntVal(field.HnswParams.M)
			}
			fieldBody.SetAttributeValue("hnsw_params", cty.ObjectVal(hnswVals))
		}
	}

	// Note: metadata is stored as a JSON string in HCL
	// For generated HCL, we skip metadata since it's complex JSON

	if c.VoiceQueryModel != "" {
		body.SetAttributeValue("voice_query_model", cty.StringVal(c.VoiceQueryModel))
	}

	return block
}

// generateSynonymBlock creates an HCL block for a synonym resource
func generateSynonymBlock(s *client.Synonym, collectionResourceName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceSynonym), resourceName})
	body := block.Body()

	// Reference the collection resource
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 9, Bytes: []byte("collection")}, // TokenIdent
		{Type: 11, Bytes: []byte(" = ")},       // TokenEqual with spaces
		{Type: 9, Bytes: []byte(fmt.Sprintf("%s.%s.name", tfnames.FullTypeName(tfnames.ResourceCollection), collectionResourceName))},
		{Type: 10, Bytes: []byte("\n")}, // TokenNewline
	})

	appendSynonymAttributes(body, s)
	return block
}

func generateSynonymBlockWithCollectionLiteral(s *client.Synonym, collectionName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceSynonym), resourceName})
	body := block.Body()

	body.SetAttributeValue("collection", cty.StringVal(collectionName))
	appendSynonymAttributes(body, s)
	return block
}

func appendSynonymAttributes(body *hclwrite.Body, s *client.Synonym) {
	body.SetAttributeValue("name", cty.StringVal(s.ID))

	if s.Root != "" {
		body.SetAttributeValue("root", cty.StringVal(s.Root))
	}

	if len(s.Synonyms) > 0 {
		vals := make([]cty.Value, len(s.Synonyms))
		for i, v := range s.Synonyms {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("synonyms", cty.ListVal(vals))
	}
}

// generateOverrideBlock creates an HCL block for an override resource
func generateOverrideBlock(o *client.Override, collectionResourceName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceOverride), resourceName})
	body := block.Body()

	// Reference the collection resource
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 9, Bytes: []byte("collection")},
		{Type: 11, Bytes: []byte(" = ")},
		{Type: 9, Bytes: []byte(fmt.Sprintf("%s.%s.name", tfnames.FullTypeName(tfnames.ResourceCollection), collectionResourceName))},
		{Type: 10, Bytes: []byte("\n")},
	})

	appendOverrideAttributes(body, o)
	return block
}

func generateOverrideBlockWithCollectionLiteral(o *client.Override, collectionName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceOverride), resourceName})
	body := block.Body()

	body.SetAttributeValue("collection", cty.StringVal(collectionName))
	appendOverrideAttributes(body, o)
	return block
}

func appendOverrideAttributes(body *hclwrite.Body, o *client.Override) {
	body.SetAttributeValue("name", cty.StringVal(o.ID))

	ruleVals := make(map[string]cty.Value)
	if o.Rule.Query != "" {
		ruleVals["query"] = cty.StringVal(o.Rule.Query)
	}
	if o.Rule.Match != "" {
		ruleVals["match"] = cty.StringVal(o.Rule.Match)
	}
	if len(o.Rule.Tags) > 0 {
		vals := make([]cty.Value, len(o.Rule.Tags))
		for i, v := range o.Rule.Tags {
			vals[i] = cty.StringVal(v)
		}
		ruleVals["tags"] = cty.ListVal(vals)
	}
	body.SetAttributeValue("rule", cty.ObjectVal(ruleVals))

	// Includes
	for _, inc := range o.Includes {
		incBlock := body.AppendNewBlock("includes", nil)
		incBody := incBlock.Body()
		incBody.SetAttributeValue("id", cty.StringVal(inc.ID))
		incBody.SetAttributeValue("position", cty.NumberIntVal(int64(inc.Position)))
	}

	// Excludes
	for _, exc := range o.Excludes {
		excBlock := body.AppendNewBlock("excludes", nil)
		excBody := excBlock.Body()
		excBody.SetAttributeValue("id", cty.StringVal(exc.ID))
	}

	// Optional fields
	if o.FilterBy != "" {
		body.SetAttributeValue("filter_by", cty.StringVal(o.FilterBy))
	}
	if o.SortBy != "" {
		body.SetAttributeValue("sort_by", cty.StringVal(o.SortBy))
	}
	if o.ReplaceQuery != "" {
		body.SetAttributeValue("replace_query", cty.StringVal(o.ReplaceQuery))
	}
	if o.RemoveMatchedTokens {
		body.SetAttributeValue("remove_matched_tokens", cty.BoolVal(true))
	} else if o.ReplaceQuery != "" {
		// Typesense rejects requests that combine replace_query with
		// remove_matched_tokens=true, so emit the false value explicitly
		// to override the schema default of true on round-trip.
		body.SetAttributeValue("remove_matched_tokens", cty.BoolVal(false))
	}
	if o.FilterCuratedHits {
		body.SetAttributeValue("filter_curated_hits", cty.BoolVal(true))
	}
	if o.StopProcessing {
		body.SetAttributeValue("stop_processing", cty.BoolVal(true))
	}
	if o.EffectiveFromTs > 0 {
		body.SetAttributeValue("effective_from_ts", cty.NumberIntVal(o.EffectiveFromTs))
	}
	if o.EffectiveToTs > 0 {
		body.SetAttributeValue("effective_to_ts", cty.NumberIntVal(o.EffectiveToTs))
	}
}

// generateStopwordsBlock creates an HCL block for a stopwords set resource
func generateStopwordsBlock(sw *client.StopwordsSet, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceStopwordsSet), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(sw.ID))

	if len(sw.Stopwords) > 0 {
		vals := make([]cty.Value, len(sw.Stopwords))
		for i, v := range sw.Stopwords {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("stopwords", cty.ListVal(vals))
	}

	if sw.Locale != "" {
		body.SetAttributeValue("locale", cty.StringVal(sw.Locale))
	}

	return block
}

// generateCollectionAliasBlock creates an HCL block for a collection alias resource
func generateCollectionAliasBlock(alias *client.CollectionAlias, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceCollectionAlias), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(alias.Name))
	body.SetAttributeValue("collection_name", cty.StringVal(alias.CollectionName))

	return block
}

// generatePresetBlock creates an HCL block for a search preset resource
func generatePresetBlock(preset *client.Preset, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourcePreset), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(preset.Name))
	if preset.Value != nil {
		valueJSON, err := json.Marshal(preset.Value)
		if err == nil {
			body.SetAttributeValue("value", cty.StringVal(string(valueJSON)))
		}
	}

	return block
}

// generateStemmingDictionaryBlock creates an HCL block for a stemming dictionary resource
func generateStemmingDictionaryBlock(dictionary *client.StemmingDictionary, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceStemmingDictionary), resourceName})
	body := block.Body()

	body.SetAttributeValue("dictionary_id", cty.StringVal(dictionary.ID))

	wordType := map[string]cty.Type{
		"word": cty.String,
		"stem": cty.String,
	}
	if len(dictionary.Words) == 0 {
		body.SetAttributeValue("words", cty.ListValEmpty(cty.Object(wordType)))
		return block
	}

	values := make([]cty.Value, len(dictionary.Words))
	for i, word := range dictionary.Words {
		values[i] = cty.ObjectVal(map[string]cty.Value{
			"word": cty.StringVal(word.Word),
			"stem": cty.StringVal(word.Stem),
		})
	}
	body.SetAttributeValue("words", cty.ListVal(values))

	return block
}

// generateClusterBlock creates an HCL block for a cloud cluster resource
func generateClusterBlock(cl *client.Cluster, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceCluster), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(cl.Name))
	body.SetAttributeValue("memory", cty.StringVal(cl.Memory))
	body.SetAttributeValue("vcpu", cty.StringVal(cl.VCPU))
	body.SetAttributeValue("high_availability", cty.StringVal(cl.HighAvailability))

	if cl.SearchDeliveryNetwork != "" {
		body.SetAttributeValue("search_delivery_network", cty.StringVal(cl.SearchDeliveryNetwork))
	}

	body.SetAttributeValue("typesense_server_version", cty.StringVal(cl.TypesenseServerVersion))

	if len(cl.Regions) > 0 {
		vals := make([]cty.Value, len(cl.Regions))
		for i, v := range cl.Regions {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("regions", cty.ListVal(vals))
	}

	if cl.AutoUpgradeCapacity {
		body.SetAttributeValue("auto_upgrade_capacity", cty.BoolVal(true))
	}

	return block
}

// generateAnalyticsRuleBlock creates an HCL block for an analytics rule resource
func generateAnalyticsRuleBlock(rule *client.AnalyticsRule, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceAnalyticsRule), resourceName})
	body := block.Body()

	body.SetAttributeValue("name", cty.StringVal(rule.Name))
	body.SetAttributeValue("type", cty.StringVal(rule.Type))

	if rule.Collection != "" {
		body.SetAttributeValue("collection", cty.StringVal(rule.Collection))
	}

	if rule.EventType != "" {
		body.SetAttributeValue("event_type", cty.StringVal(rule.EventType))
	}

	// Serialize params as JSON string
	if len(rule.Params) > 0 {
		paramsJSON, err := json.Marshal(rule.Params)
		if err == nil {
			body.SetAttributeValue("params", cty.StringVal(string(paramsJSON)))
		}
	}

	return block
}

// generateAPIKeyBlock creates an HCL block for an API key resource
func generateAPIKeyBlock(key *client.APIKey, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceAPIKey), resourceName})
	body := block.Body()

	// Add comment about non-recoverable key value
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 4, Bytes: []byte("# Note: API key value is not recoverable after creation. The imported key will have a placeholder value.\n")},
	})

	if key.Description != "" {
		body.SetAttributeValue("description", cty.StringVal(key.Description))
	}

	if len(key.Actions) > 0 {
		vals := make([]cty.Value, len(key.Actions))
		for i, v := range key.Actions {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("actions", cty.ListVal(vals))
	}

	if len(key.Collections) > 0 {
		vals := make([]cty.Value, len(key.Collections))
		for i, v := range key.Collections {
			vals[i] = cty.StringVal(v)
		}
		body.SetAttributeValue("collections", cty.ListVal(vals))
	}

	// Only include expires_at if it has a reasonable value (before year 3000)
	if key.ExpiresAt > 0 && key.ExpiresAt < 32503680000 {
		body.SetAttributeValue("expires_at", cty.NumberIntVal(key.ExpiresAt))
	}

	return block
}

// generateNLSearchModelBlock creates an HCL block for a NL search model resource
func generateNLSearchModelBlock(model *client.NLSearchModel, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceNLSearchModel), resourceName})
	body := block.Body()

	body.SetAttributeValue("id", cty.StringVal(model.ID))
	body.SetAttributeValue("model_name", cty.StringVal(model.ModelName))

	// API key is sensitive and not returned by the API - use a variable reference
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 4, Bytes: []byte("# api_key is sensitive and not recoverable from the API. Set via variable.\n")},
		{Type: 9, Bytes: []byte("api_key")},
		{Type: 11, Bytes: []byte(" = ")},
		{Type: 9, Bytes: []byte("var.openai_api_key")},
		{Type: 10, Bytes: []byte("\n")},
	})

	if model.SystemPrompt != "" {
		body.SetAttributeValue("system_prompt", cty.StringVal(model.SystemPrompt))
	}

	if model.MaxBytes > 0 {
		body.SetAttributeValue("max_bytes", cty.NumberIntVal(model.MaxBytes))
	}

	if model.Temperature != nil {
		body.SetAttributeValue("temperature", cty.NumberFloatVal(*model.Temperature))
	}

	if model.AccountID != "" {
		body.SetAttributeValue("account_id", cty.StringVal(model.AccountID))
	}

	if model.APIURL != "" {
		body.SetAttributeValue("api_url", cty.StringVal(model.APIURL))
	}

	return block
}

// generateConversationModelBlock creates an HCL block for a conversation model resource
func generateConversationModelBlock(model *client.ConversationModel, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{tfnames.FullTypeName(tfnames.ResourceConversationModel), resourceName})
	body := block.Body()

	if model.ID != "" {
		body.SetAttributeValue("id", cty.StringVal(model.ID))
	}

	body.SetAttributeValue("model_name", cty.StringVal(model.ModelName))

	// API key is sensitive and not returned by the API - use a variable reference
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 4, Bytes: []byte("# api_key is sensitive and not recoverable from the API. Set via variable.\n")},
		{Type: 9, Bytes: []byte("api_key")},
		{Type: 11, Bytes: []byte(" = ")},
		{Type: 9, Bytes: []byte("var.openai_api_key")},
		{Type: 10, Bytes: []byte("\n")},
	})

	body.SetAttributeValue("history_collection", cty.StringVal(model.HistoryCollection))
	body.SetAttributeValue("system_prompt", cty.StringVal(model.SystemPrompt))

	if model.TTL > 0 {
		body.SetAttributeValue("ttl", cty.NumberIntVal(model.TTL))
	}

	if model.MaxBytes > 0 {
		body.SetAttributeValue("max_bytes", cty.NumberIntVal(model.MaxBytes))
	}

	if model.AccountID != "" {
		body.SetAttributeValue("account_id", cty.StringVal(model.AccountID))
	}

	if model.VllmURL != "" {
		body.SetAttributeValue("vllm_url", cty.StringVal(model.VllmURL))
	}

	return block
}
