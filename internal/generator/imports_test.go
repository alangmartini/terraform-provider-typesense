package generator

import (
	"strings"
	"testing"
)

func TestGenerateImportScript(t *testing.T) {
	commands := []ImportCommand{
		{
			ResourceType: "typesense_collection",
			ResourceName: "products",
			ImportID:     "products",
		},
		{
			ResourceType: "typesense_synonym",
			ResourceName: "products_clothing",
			ImportID:     "products/clothing",
		},
	}

	script := GenerateImportScript(commands)

	// Check header
	if !strings.HasPrefix(script, "#!/bin/bash") {
		t.Error("Script should start with shebang")
	}
	if !strings.Contains(script, "set -e") {
		t.Error("Script should have error handling")
	}

	// Check import commands
	if !strings.Contains(script, `terraform import typesense_collection.products "products"`) {
		t.Error("Script should contain collection import command")
	}
	if !strings.Contains(script, `terraform import typesense_synonym.products_clothing "products/clothing"`) {
		t.Error("Script should contain synonym import command")
	}
}

func TestCollectionImportID(t *testing.T) {
	id := CollectionImportID("products")
	if id != "products" {
		t.Errorf("CollectionImportID = %q, want %q", id, "products")
	}
}

func TestSynonymImportID(t *testing.T) {
	id := SynonymImportID("products", "clothing")
	if id != "products/clothing" {
		t.Errorf("SynonymImportID = %q, want %q", id, "products/clothing")
	}
}

func TestOverrideImportID(t *testing.T) {
	id := OverrideImportID("products", "promote_sale")
	if id != "products/promote_sale" {
		t.Errorf("OverrideImportID = %q, want %q", id, "products/promote_sale")
	}
}

func TestStopwordsImportID(t *testing.T) {
	id := StopwordsImportID("common_words")
	if id != "common_words" {
		t.Errorf("StopwordsImportID = %q, want %q", id, "common_words")
	}
}

func TestClusterImportID(t *testing.T) {
	id := ClusterImportID("abc123")
	if id != "abc123" {
		t.Errorf("ClusterImportID = %q, want %q", id, "abc123")
	}
}

func TestAnalyticsRuleImportID(t *testing.T) {
	id := AnalyticsRuleImportID("popular_searches")
	if id != "popular_searches" {
		t.Errorf("AnalyticsRuleImportID = %q, want %q", id, "popular_searches")
	}
}

func TestAPIKeyImportID(t *testing.T) {
	id := APIKeyImportID(42)
	if id != "42" {
		t.Errorf("APIKeyImportID = %q, want %q", id, "42")
	}
}

func TestNLSearchModelImportID(t *testing.T) {
	id := NLSearchModelImportID("nl_model_1")
	if id != "nl_model_1" {
		t.Errorf("NLSearchModelImportID = %q, want %q", id, "nl_model_1")
	}
}

func TestConversationModelImportID(t *testing.T) {
	id := ConversationModelImportID("conv_model_1")
	if id != "conv_model_1" {
		t.Errorf("ConversationModelImportID = %q, want %q", id, "conv_model_1")
	}
}

func TestImportScriptWithNewResourceTypes(t *testing.T) {
	commands := []ImportCommand{
		{
			ResourceType: "typesense_analytics_rule",
			ResourceName: "popular_searches",
			ImportID:     "popular_searches",
		},
		{
			ResourceType: "typesense_api_key",
			ResourceName: "search_only_key",
			ImportID:     "1",
		},
		{
			ResourceType: "typesense_nl_search_model",
			ResourceName: "nl_model_1",
			ImportID:     "nl_model_1",
		},
		{
			ResourceType: "typesense_conversation_model",
			ResourceName: "conv_model_1",
			ImportID:     "conv_model_1",
		},
	}

	script := GenerateImportScript(commands)

	if !strings.Contains(script, `terraform import typesense_analytics_rule.popular_searches "popular_searches"`) {
		t.Error("Script should contain analytics rule import command")
	}
	if !strings.Contains(script, `terraform import typesense_api_key.search_only_key "1"`) {
		t.Error("Script should contain API key import command")
	}
	if !strings.Contains(script, `terraform import typesense_nl_search_model.nl_model_1 "nl_model_1"`) {
		t.Error("Script should contain NL search model import command")
	}
	if !strings.Contains(script, `terraform import typesense_conversation_model.conv_model_1 "conv_model_1"`) {
		t.Error("Script should contain conversation model import command")
	}
}
