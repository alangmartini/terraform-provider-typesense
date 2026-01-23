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
				Sort:   true,
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

func TestGenerateAPIKeyBlock(t *testing.T) {
	apiKey := &client.APIKey{
		ID:          42,
		Description: "Search-only key",
		Actions:     []string{"documents:search"},
		Collections: []string{"products", "users"},
		ExpiresAt:   1735689600,
	}

	block := generateAPIKeyBlock(apiKey, "key_42")
	hcl := blockToHCL(block)

	if !strings.Contains(hcl, `resource "typesense_api_key" "key_42"`) {
		t.Error("Block should contain resource type and name")
	}
	if !containsAttr(hcl, "description", `"Search-only key"`) {
		t.Error("Block should contain description")
	}
	if !strings.Contains(hcl, `"documents:search"`) {
		t.Error("Block should contain actions")
	}
}

func TestGenerateAPIKeyComment(t *testing.T) {
	comment := generateAPIKeyComment(42)

	if !strings.Contains(comment, "WARNING") {
		t.Error("Comment should contain warning")
	}
	if !strings.Contains(comment, "key ID: 42") {
		t.Error("Comment should contain key ID")
	}
	if !strings.Contains(comment, "terraform import typesense_api_key.key_42 42") {
		t.Error("Comment should contain import command")
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
