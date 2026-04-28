package generator

import (
	"strings"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
)

func TestGenerateImportBlocks(t *testing.T) {
	commands := []ImportCommand{
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceCollection),
			ResourceName: "products",
			ImportID:     "products",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceSynonym),
			ResourceName: "products_clothing",
			ImportID:     "products/clothing",
		},
	}

	f := GenerateImportBlocks(commands)
	output := string(f.Bytes())

	// Check header comment
	if !strings.Contains(output, "Generated Terraform import blocks") {
		t.Error("Output should contain header comment")
	}

	// Check collection import block
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceCollection)+".products") {
		t.Error("Output should contain collection import block 'to' reference")
	}
	if !strings.Contains(output, `id = "products"`) {
		t.Error("Output should contain collection import ID")
	}

	// Check synonym import block
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceSynonym)+".products_clothing") {
		t.Error("Output should contain synonym import block 'to' reference")
	}
	if !strings.Contains(output, `id = "products/clothing"`) {
		t.Error("Output should contain synonym import ID")
	}

	// Check that it uses import blocks, not terraform import commands
	if strings.Contains(output, "terraform import") {
		t.Error("Output should use import blocks, not terraform import CLI commands")
	}
	if strings.Contains(output, "#!/bin/bash") {
		t.Error("Output should not be a bash script")
	}

	// Count import blocks
	importCount := strings.Count(output, "import {")
	if importCount != 2 {
		t.Errorf("Expected 2 import blocks, got %d", importCount)
	}
}

func TestGenerateImportBlocksEmpty(t *testing.T) {
	f := GenerateImportBlocks(nil)
	output := string(f.Bytes())

	if strings.Contains(output, "import {") {
		t.Error("Empty commands should produce no import blocks")
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

func TestCollectionAliasImportID(t *testing.T) {
	id := CollectionAliasImportID("music")
	if id != "music" {
		t.Errorf("CollectionAliasImportID = %q, want %q", id, "music")
	}
}

func TestPresetImportID(t *testing.T) {
	id := PresetImportID("track-listing")
	if id != "track-listing" {
		t.Errorf("PresetImportID = %q, want %q", id, "track-listing")
	}
}

func TestStemmingDictionaryImportID(t *testing.T) {
	id := StemmingDictionaryImportID("music-terms")
	if id != "music-terms" {
		t.Errorf("StemmingDictionaryImportID = %q, want %q", id, "music-terms")
	}
}

func TestImportBlocksWithNewResourceTypes(t *testing.T) {
	commands := []ImportCommand{
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceAnalyticsRule),
			ResourceName: "popular_searches",
			ImportID:     "popular_searches",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceAPIKey),
			ResourceName: "search_only_key",
			ImportID:     "1",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceNLSearchModel),
			ResourceName: "nl_model_1",
			ImportID:     "nl_model_1",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceConversationModel),
			ResourceName: "conv_model_1",
			ImportID:     "conv_model_1",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceCollectionAlias),
			ResourceName: "music",
			ImportID:     "music",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourcePreset),
			ResourceName: "track_listing",
			ImportID:     "track-listing",
		},
		{
			ResourceType: tfnames.FullTypeName(tfnames.ResourceStemmingDictionary),
			ResourceName: "music_terms",
			ImportID:     "music-terms",
		},
	}

	f := GenerateImportBlocks(commands)
	output := string(f.Bytes())

	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceAnalyticsRule)+".popular_searches") {
		t.Error("Output should contain analytics rule import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceAPIKey)+".search_only_key") {
		t.Error("Output should contain API key import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceNLSearchModel)+".nl_model_1") {
		t.Error("Output should contain NL search model import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceConversationModel)+".conv_model_1") {
		t.Error("Output should contain conversation model import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceCollectionAlias)+".music") {
		t.Error("Output should contain collection alias import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourcePreset)+".track_listing") {
		t.Error("Output should contain preset import block")
	}
	if !strings.Contains(output, tfnames.FullTypeName(tfnames.ResourceStemmingDictionary)+".music_terms") {
		t.Error("Output should contain stemming dictionary import block")
	}

	importCount := strings.Count(output, "import {")
	if importCount != 7 {
		t.Errorf("Expected 7 import blocks, got %d", importCount)
	}
}
