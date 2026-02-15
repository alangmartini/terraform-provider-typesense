package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// blockToHCL converts an hclwrite.Block to its HCL string representation
func blockToHCL(block *hclwrite.Block) string {
	f := hclwrite.NewEmptyFile()
	f.Body().AppendBlock(block)
	return string(f.Bytes())
}

// containsAttr checks if HCL contains an attribute with given name and value
// Handles HCL's variable whitespace around the equals sign
func containsAttr(hcl, attrName, attrValue string) bool {
	// Match attribute = value with flexible whitespace
	pattern := regexp.MustCompile(regexp.QuoteMeta(attrName) + `\s*=\s*` + regexp.QuoteMeta(attrValue))
	return pattern.MatchString(hcl)
}

func TestGenerateCollectionBlock(t *testing.T) {
	indexFalse := false
	sortTrue := true
	collection := &client.Collection{
		Name:                "products",
		DefaultSortingField: "popularity",
		EnableNestedFields:  true,
		TokenSeparators:     []string{"-", "_"},
		Fields: []client.CollectionField{
			{
				Name:  "id",
				Type:  "string",
				Index: &indexFalse,
			},
			{
				Name:     "name",
				Type:     "string",
				Facet:    true,
				Optional: true,
			},
			{
				Name:   "price",
				Type:   "float",
				Sort:   &sortTrue,
				Locale: "en",
			},
		},
	}

	block := generateCollectionBlock(collection, "products")
	hcl := blockToHCL(block)

	// Check essential parts are present
	if !strings.Contains(hcl, `resource "typesense_collection" "products"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "name", `"products"`) {
		t.Error("Block should contain name attribute")
	}
	if !containsAttr(hcl, "default_sorting_field", `"popularity"`) {
		t.Error("Block should contain default_sorting_field")
	}
	if !containsAttr(hcl, "enable_nested_fields", "true") {
		t.Error("Block should contain enable_nested_fields")
	}
	if !strings.Contains(hcl, "field {") {
		t.Error("Block should contain field blocks")
	}
}

func TestGenerateSynonymBlock(t *testing.T) {
	synonym := &client.Synonym{
		ID:       "clothing",
		Synonyms: []string{"shirt", "tee", "top"},
	}

	block := generateSynonymBlock(synonym, "products", "products_clothing")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_synonym" "products_clothing"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "name", `"clothing"`) {
		t.Error("Block should contain name attribute")
	}
	if !strings.Contains(hcl, "typesense_collection.products.name") {
		t.Error("Block should reference collection")
	}
}

func TestGenerateSynonymBlockWithRoot(t *testing.T) {
	synonym := &client.Synonym{
		ID:       "blazer",
		Root:     "jacket",
		Synonyms: []string{"blazer", "coat"},
	}

	block := generateSynonymBlock(synonym, "products", "products_blazer")
	hcl := blockToHCL(block)

	if !containsAttr(hcl, "root", `"jacket"`) {
		t.Error("Block should contain root attribute for one-way synonym")
	}
}

func TestGenerateOverrideBlock(t *testing.T) {
	override := &client.Override{
		ID: "promote_sale",
		Rule: client.OverrideRule{
			Query: "sale",
			Match: "exact",
		},
		Includes: []client.OverrideInclude{
			{ID: "doc1", Position: 1},
			{ID: "doc2", Position: 2},
		},
		Excludes: []client.OverrideExclude{
			{ID: "doc3"},
		},
		FilterBy:            "category:electronics",
		RemoveMatchedTokens: true,
	}

	block := generateOverrideBlock(override, "products", "products_promote_sale")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_override" "products_promote_sale"`) {
		t.Error("Block should contain resource type and name")
	}
	if !strings.Contains(hcl, "rule {") {
		t.Error("Block should contain rule block")
	}
	if !containsAttr(hcl, "query", `"sale"`) {
		t.Error("Block should contain query in rule")
	}
	if !strings.Contains(hcl, "includes {") {
		t.Error("Block should contain includes blocks")
	}
	if !strings.Contains(hcl, "excludes {") {
		t.Error("Block should contain excludes blocks")
	}
	if !containsAttr(hcl, "filter_by", `"category:electronics"`) {
		t.Error("Block should contain filter_by")
	}
}

func TestGenerateStopwordsBlock(t *testing.T) {
	stopwords := &client.StopwordsSet{
		ID:        "common_words",
		Stopwords: []string{"the", "a", "an"},
		Locale:    "en",
	}

	block := generateStopwordsBlock(stopwords, "common_words")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_stopwords" "common_words"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "name", `"common_words"`) {
		t.Error("Block should contain name attribute")
	}
	if !containsAttr(hcl, "locale", `"en"`) {
		t.Error("Block should contain locale")
	}
}

func TestGenerateClusterBlock(t *testing.T) {
	cluster := &client.Cluster{
		ID:                     "abc123",
		Name:                   "my-cluster",
		Memory:                 "0.5_gb",
		VCPU:                   "2_vcpu_1_hr_burst",
		HighAvailability:       "false",
		TypesenseServerVersion: "28.0",
		Regions:                []string{"us-west-2"},
		AutoUpgradeCapacity:    true,
	}

	block := generateClusterBlock(cluster, "my_cluster")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_cluster" "my_cluster"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "name", `"my-cluster"`) {
		t.Error("Block should contain name")
	}
	if !containsAttr(hcl, "memory", `"0.5_gb"`) {
		t.Error("Block should contain memory")
	}
	if !containsAttr(hcl, "auto_upgrade_capacity", "true") {
		t.Error("Block should contain auto_upgrade_capacity")
	}
}

func TestGenerateAnalyticsRuleBlock(t *testing.T) {
	rule := &client.AnalyticsRule{
		Name:       "popular_searches",
		Type:       "popular_queries",
		Collection: "products",
		EventType:  "search",
		Params: map[string]any{
			"destination_collection": "product_queries",
			"limit":                  float64(1000),
		},
	}

	block := generateAnalyticsRuleBlock(rule, "popular_searches")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_analytics_rule" "popular_searches"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "name", `"popular_searches"`) {
		t.Error("Block should contain name attribute")
	}
	if !containsAttr(hcl, "type", `"popular_queries"`) {
		t.Error("Block should contain type attribute")
	}
	if !containsAttr(hcl, "collection", `"products"`) {
		t.Error("Block should contain collection attribute")
	}
	if !containsAttr(hcl, "event_type", `"search"`) {
		t.Error("Block should contain event_type attribute")
	}
	if !strings.Contains(hcl, "destination_collection") {
		t.Error("Block should contain params with destination_collection")
	}
}

func TestGenerateAnalyticsRuleBlockCounter(t *testing.T) {
	rule := &client.AnalyticsRule{
		Name:       "click_counter",
		Type:       "counter",
		Collection: "products",
		EventType:  "click",
		Params: map[string]any{
			"destination_collection": "product_clicks",
			"counter_field":         "click_count",
		},
	}

	block := generateAnalyticsRuleBlock(rule, "click_counter")
	hcl := blockToHCL(block)

	if !containsAttr(hcl, "type", `"counter"`) {
		t.Error("Block should contain counter type")
	}
	if !containsAttr(hcl, "event_type", `"click"`) {
		t.Error("Block should contain click event_type")
	}
}

func TestGenerateAPIKeyBlock(t *testing.T) {
	key := &client.APIKey{
		ID:          1,
		Description: "Search-only key",
		Actions:     []string{"documents:search"},
		Collections: []string{"products", "categories"},
		ExpiresAt:   1735689600,
	}

	block := generateAPIKeyBlock(key, "search_only_key")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_api_key" "search_only_key"`) {
		t.Error("Block should contain resource type and name")
	}
	if !strings.Contains(hcl, "API key value is not recoverable") {
		t.Error("Block should contain comment about non-recoverable key")
	}
	if !containsAttr(hcl, "description", `"Search-only key"`) {
		t.Error("Block should contain description")
	}
	if !strings.Contains(hcl, `"documents:search"`) {
		t.Error("Block should contain actions")
	}
	if !strings.Contains(hcl, `"products"`) {
		t.Error("Block should contain collections")
	}
	if !strings.Contains(hcl, "1735689600") {
		t.Error("Block should contain expires_at")
	}
}

func TestGenerateAPIKeyBlockNoExpiry(t *testing.T) {
	key := &client.APIKey{
		ID:          2,
		Description: "Admin key",
		Actions:     []string{"*"},
		Collections: []string{"*"},
		ExpiresAt:   64723363199, // Far-future default
	}

	block := generateAPIKeyBlock(key, "admin_key")
	hcl := blockToHCL(block)

	// Far-future expiry should be omitted
	if strings.Contains(hcl, "expires_at") {
		t.Error("Block should not contain expires_at for far-future default")
	}
}

func TestGenerateNLSearchModelBlock(t *testing.T) {
	temp := 0.5
	model := &client.NLSearchModel{
		ID:           "nl_model_1",
		ModelName:    "openai/gpt-4o-mini",
		SystemPrompt: "You are a search assistant.",
		MaxBytes:     16000,
		Temperature:  &temp,
	}

	block := generateNLSearchModelBlock(model, "nl_model_1")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_nl_search_model" "nl_model_1"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "id", `"nl_model_1"`) {
		t.Error("Block should contain id attribute")
	}
	if !containsAttr(hcl, "model_name", `"openai/gpt-4o-mini"`) {
		t.Error("Block should contain model_name")
	}
	if !strings.Contains(hcl, "var.openai_api_key") {
		t.Error("Block should reference var.openai_api_key for api_key")
	}
	if !strings.Contains(hcl, "api_key is sensitive") {
		t.Error("Block should contain comment about sensitive api_key")
	}
	if !containsAttr(hcl, "system_prompt", `"You are a search assistant."`) {
		t.Error("Block should contain system_prompt")
	}
	if !strings.Contains(hcl, "16000") {
		t.Error("Block should contain max_bytes")
	}
}

func TestGenerateConversationModelBlock(t *testing.T) {
	model := &client.ConversationModel{
		ID:                "conv_model_1",
		ModelName:         "openai/gpt-4o",
		HistoryCollection: "conversation_history",
		SystemPrompt:      "You are a helpful assistant.",
		TTL:               86400,
		MaxBytes:          32000,
	}

	block := generateConversationModelBlock(model, "conv_model_1")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_conversation_model" "conv_model_1"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "id", `"conv_model_1"`) {
		t.Error("Block should contain id attribute")
	}
	if !containsAttr(hcl, "model_name", `"openai/gpt-4o"`) {
		t.Error("Block should contain model_name")
	}
	if !strings.Contains(hcl, "var.openai_api_key") {
		t.Error("Block should reference var.openai_api_key for api_key")
	}
	if !containsAttr(hcl, "history_collection", `"conversation_history"`) {
		t.Error("Block should contain history_collection")
	}
	if !containsAttr(hcl, "system_prompt", `"You are a helpful assistant."`) {
		t.Error("Block should contain system_prompt")
	}
	if !strings.Contains(hcl, "86400") {
		t.Error("Block should contain ttl")
	}
	if !strings.Contains(hcl, "32000") {
		t.Error("Block should contain max_bytes")
	}
}

func TestGenerateConversationModelBlockWithVllm(t *testing.T) {
	model := &client.ConversationModel{
		ID:                "vllm_model",
		ModelName:         "meta/llama-3-8b-instruct",
		HistoryCollection: "chat_history",
		SystemPrompt:      "Answer questions.",
		VllmURL:           "http://localhost:8000",
	}

	block := generateConversationModelBlock(model, "vllm_model")
	hcl := blockToHCL(block)

	if !containsAttr(hcl, "vllm_url", `"http://localhost:8000"`) {
		t.Error("Block should contain vllm_url")
	}
}
