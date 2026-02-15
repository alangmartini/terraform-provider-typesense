package generator

import (
	"fmt"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// generateTerraformBlock creates the terraform required_providers block
func generateTerraformBlock(f *hclwrite.File) {
	tfBlock := f.Body().AppendNewBlock("terraform", nil)
	reqProviders := tfBlock.Body().AppendNewBlock("required_providers", nil)
	reqProviders.Body().SetAttributeValue("typesense", cty.ObjectVal(map[string]cty.Value{
		"source": cty.StringVal("alanm/typesense"),
	}))
	f.Body().AppendNewline()
}

// generateProviderBlock creates the provider configuration block
func generateProviderBlock(f *hclwrite.File, host string, port int, protocol string) {
	providerBlock := f.Body().AppendNewBlock("provider", []string{"typesense"})
	providerBlock.Body().SetAttributeValue("server_host", cty.StringVal(host))
	providerBlock.Body().SetAttributeValue("server_port", cty.NumberIntVal(int64(port)))
	providerBlock.Body().SetAttributeValue("server_protocol", cty.StringVal(protocol))
	// Add comment for API key
	providerBlock.Body().AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 4, Bytes: []byte("# server_api_key = \"YOUR_API_KEY_HERE\"\n")}, // TokenComment = 4
	})
	f.Body().AppendNewline()
}

// generateCollectionBlock creates an HCL block for a collection resource
func generateCollectionBlock(c *client.Collection, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"typesense_collection", resourceName})
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
		if field.AsyncReference != "" {
			fieldBody.SetAttributeValue("async_reference", cty.StringVal(field.AsyncReference))
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
			embedBlock := fieldBody.AppendNewBlock("embed", nil)
			embedBody := embedBlock.Body()
			if len(field.Embed.From) > 0 {
				fromVals := make([]cty.Value, len(field.Embed.From))
				for i, v := range field.Embed.From {
					fromVals[i] = cty.StringVal(v)
				}
				embedBody.SetAttributeValue("from", cty.ListVal(fromVals))
			}
			mcBlock := embedBody.AppendNewBlock("model_config", nil)
			mcBody := mcBlock.Body()
			mcBody.SetAttributeValue("model_name", cty.StringVal(field.Embed.ModelConfig.ModelName))
			if field.Embed.ModelConfig.URL != "" {
				mcBody.SetAttributeValue("url", cty.StringVal(field.Embed.ModelConfig.URL))
			}
			// Intentionally omit api_key from generated HCL (sensitive)
		}
		if field.HnswParams != nil {
			hpBlock := fieldBody.AppendNewBlock("hnsw_params", nil)
			hpBody := hpBlock.Body()
			if field.HnswParams.EfConstruction > 0 {
				hpBody.SetAttributeValue("ef_construction", cty.NumberIntVal(field.HnswParams.EfConstruction))
			}
			if field.HnswParams.M > 0 {
				hpBody.SetAttributeValue("m", cty.NumberIntVal(field.HnswParams.M))
			}
		}
	}

	// Collection-level metadata
	if c.Metadata != nil {
		// Note: metadata is stored as a JSON string in HCL
		// For generated HCL, we skip metadata since it's complex JSON
	}
	if c.VoiceQueryModel != "" {
		body.SetAttributeValue("voice_query_model", cty.StringVal(c.VoiceQueryModel))
	}

	return block
}

// generateSynonymBlock creates an HCL block for a synonym resource
func generateSynonymBlock(s *client.Synonym, collectionResourceName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"typesense_synonym", resourceName})
	body := block.Body()

	// Reference the collection resource
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 9, Bytes: []byte("collection")}, // TokenIdent
		{Type: 11, Bytes: []byte(" = ")},        // TokenEqual with spaces
		{Type: 9, Bytes: []byte(fmt.Sprintf("typesense_collection.%s.name", collectionResourceName))},
		{Type: 10, Bytes: []byte("\n")}, // TokenNewline
	})

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

	return block
}

// generateOverrideBlock creates an HCL block for an override resource
func generateOverrideBlock(o *client.Override, collectionResourceName, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"typesense_override", resourceName})
	body := block.Body()

	// Reference the collection resource
	body.AppendUnstructuredTokens(hclwrite.Tokens{
		{Type: 9, Bytes: []byte("collection")},
		{Type: 11, Bytes: []byte(" = ")},
		{Type: 9, Bytes: []byte(fmt.Sprintf("typesense_collection.%s.name", collectionResourceName))},
		{Type: 10, Bytes: []byte("\n")},
	})

	body.SetAttributeValue("name", cty.StringVal(o.ID))

	// Rule block
	ruleBlock := body.AppendNewBlock("rule", nil)
	ruleBody := ruleBlock.Body()
	if o.Rule.Query != "" {
		ruleBody.SetAttributeValue("query", cty.StringVal(o.Rule.Query))
	}
	if o.Rule.Match != "" {
		ruleBody.SetAttributeValue("match", cty.StringVal(o.Rule.Match))
	}
	if len(o.Rule.Tags) > 0 {
		vals := make([]cty.Value, len(o.Rule.Tags))
		for i, v := range o.Rule.Tags {
			vals[i] = cty.StringVal(v)
		}
		ruleBody.SetAttributeValue("tags", cty.ListVal(vals))
	}

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

	return block
}

// generateStopwordsBlock creates an HCL block for a stopwords set resource
func generateStopwordsBlock(sw *client.StopwordsSet, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"typesense_stopwords", resourceName})
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

// generateClusterBlock creates an HCL block for a cloud cluster resource
func generateClusterBlock(cl *client.Cluster, resourceName string) *hclwrite.Block {
	block := hclwrite.NewBlock("resource", []string{"typesense_cluster", resourceName})
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
