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
		{
			ResourceType: "typesense_api_key",
			ResourceName: "key_42",
			ImportID:     "42",
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
	if !strings.Contains(script, `terraform import typesense_api_key.key_42 "42"`) {
		t.Error("Script should contain API key import command")
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

func TestAPIKeyImportID(t *testing.T) {
	id := APIKeyImportID(42)
	if id != "42" {
		t.Errorf("APIKeyImportID = %q, want %q", id, "42")
	}
}

func TestClusterImportID(t *testing.T) {
	id := ClusterImportID("abc123")
	if id != "abc123" {
		t.Errorf("ClusterImportID = %q, want %q", id, "abc123")
	}
}
