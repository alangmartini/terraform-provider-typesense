package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// API Payload Validation Tests
// =============================================================================
// These tests validate that our Go structs serialize to JSON payloads that match
// what the Typesense API expects. This catches field naming mismatches early
// without requiring a live Typesense server.

// TestSynonymSetJSONSerialization validates that SynonymSet serializes to the
// format expected by the Typesense v30 API. The API expects an "items" field
// containing the array of synonym rules, not "synonyms".
func TestSynonymSetJSONSerialization(t *testing.T) {
	synonymSet := SynonymSet{
		Name: "test-synonyms",
		Synonyms: []SynonymItem{
			{
				ID:       "movie-synonyms",
				Synonyms: []string{"film", "movie", "picture"},
			},
			{
				ID:       "laptop-synonyms",
				Root:     "laptop",
				Synonyms: []string{"notebook", "portable computer"},
			},
		},
	}

	data, err := json.Marshal(synonymSet)
	if err != nil {
		t.Fatalf("Failed to marshal SynonymSet: %v", err)
	}

	// Parse the JSON to verify field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify "name" field exists
	if _, ok := result["name"]; !ok {
		t.Error("Expected 'name' field in JSON output")
	}

	// CRITICAL: Verify API expects "items" field, NOT "synonyms"
	// This is the exact issue we're testing for - Typesense v30 API requires "items"
	if _, ok := result["items"]; !ok {
		t.Error("Expected 'items' field in JSON output - Typesense v30 API requires 'items' not 'synonyms'")
	}

	// Verify "synonyms" is NOT used as the top-level field name
	// (though each item inside still has its own "synonyms" array)
	if _, ok := result["synonyms"]; ok {
		t.Error("Top-level 'synonyms' field should not exist - API expects 'items'")
	}

	// Verify the items array structure
	items, ok := result["items"].([]interface{})
	if !ok {
		t.Fatal("'items' field is not an array")
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// Verify first item structure
	firstItem, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatal("First item is not an object")
	}

	if firstItem["id"] != "movie-synonyms" {
		t.Errorf("Expected first item id 'movie-synonyms', got %v", firstItem["id"])
	}

	// Each SynonymItem still has its own "synonyms" field (the actual synonym words)
	if _, ok := firstItem["synonyms"]; !ok {
		t.Error("Expected 'synonyms' field within each item")
	}
}

// TestSynonymItemJSONSerialization validates that individual synonym items
// serialize correctly with their "synonyms" field for the actual synonym words.
func TestSynonymItemJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		item     SynonymItem
		wantRoot bool
	}{
		{
			name: "multi-way synonym",
			item: SynonymItem{
				ID:       "colors",
				Synonyms: []string{"red", "crimson", "scarlet"},
			},
			wantRoot: false,
		},
		{
			name: "one-way synonym with root",
			item: SynonymItem{
				ID:       "laptop",
				Root:     "laptop",
				Synonyms: []string{"notebook", "netbook"},
			},
			wantRoot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.item)
			if err != nil {
				t.Fatalf("Failed to marshal SynonymItem: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			// Verify required fields
			if _, ok := result["id"]; !ok {
				t.Error("Expected 'id' field")
			}
			if _, ok := result["synonyms"]; !ok {
				t.Error("Expected 'synonyms' field")
			}

			// Verify root field presence
			_, hasRoot := result["root"]
			if tt.wantRoot && !hasRoot {
				t.Error("Expected 'root' field for one-way synonym")
			}
			if !tt.wantRoot && hasRoot {
				t.Error("Did not expect 'root' field for multi-way synonym")
			}
		})
	}
}

// TestCurationSetJSONSerialization validates that CurationSet serializes correctly.
func TestCurationSetJSONSerialization(t *testing.T) {
	curationSet := CurationSet{
		Name: "test-curations",
		Curations: []CurationItem{
			{
				ID: "featured-products",
				Rule: OverrideRule{
					Query: "laptop",
					Match: "exact",
				},
				Includes: []OverrideInclude{
					{ID: "product-123", Position: 1},
				},
			},
		},
	}

	data, err := json.Marshal(curationSet)
	if err != nil {
		t.Fatalf("Failed to marshal CurationSet: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify "name" field exists
	if _, ok := result["name"]; !ok {
		t.Error("Expected 'name' field in JSON output")
	}

	// CRITICAL: Verify API expects "items" field, NOT "curations"
	// This is the exact issue that caused "Missing or invalid 'items' field" error
	if _, ok := result["items"]; !ok {
		t.Error("Expected 'items' field in JSON output - Typesense v30 API requires 'items' not 'curations'")
	}

	// Verify "curations" is NOT used as the field name
	if _, ok := result["curations"]; ok {
		t.Error("'curations' field should not exist - API expects 'items'")
	}
}

// TestEmptySynonymSetSerialization verifies that an empty synonym set
// always includes the "items" field (required by the Typesense API).
func TestEmptySynonymSetSerialization(t *testing.T) {
	synonymSet := SynonymSet{
		Name:     "empty-set",
		Synonyms: nil, // empty
	}

	data, err := json.Marshal(synonymSet)
	if err != nil {
		t.Fatalf("Failed to marshal SynonymSet: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// The "items" field must always be present (Typesense returns 400 without it)
	if _, ok := result["items"]; !ok {
		t.Error("Expected 'items' field to be present even when empty (required by Typesense API)")
	}
}

// =============================================================================
// Collection API Payload Tests
// =============================================================================

func TestCollectionJSONSerialization(t *testing.T) {
	indexTrue := true
	sortFalse := false

	collection := Collection{
		Name:                "products",
		DefaultSortingField: "popularity",
		TokenSeparators:     []string{"-", "_"},
		SymbolsToIndex:      []string{"+", "&"},
		EnableNestedFields:  true,
		Fields: []CollectionField{
			{
				Name:     "title",
				Type:     "string",
				Facet:    true,
				Optional: false,
				Index:    &indexTrue,
			},
			{
				Name:  "price",
				Type:  "float",
				Sort:  &sortFalse,
				Index: &indexTrue,
			},
		},
	}

	data, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("Failed to marshal Collection: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify required fields
	expectedFields := []string{"name", "fields"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in Collection JSON", field)
		}
	}

	// Verify optional fields when set
	optionalFields := []string{"default_sorting_field", "token_separators", "symbols_to_index", "enable_nested_fields"}
	for _, field := range optionalFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in Collection JSON when set", field)
		}
	}

	// Verify fields array structure
	fields, ok := result["fields"].([]interface{})
	if !ok {
		t.Fatal("'fields' is not an array")
	}

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}

	// Check first field structure
	firstField := fields[0].(map[string]interface{})
	if firstField["name"] != "title" {
		t.Errorf("Expected field name 'title', got %v", firstField["name"])
	}
	if firstField["type"] != "string" {
		t.Errorf("Expected field type 'string', got %v", firstField["type"])
	}
}

// =============================================================================
// Synonym (v29 per-collection) API Payload Tests
// =============================================================================

func TestSynonymJSONSerialization(t *testing.T) {
	tests := []struct {
		name           string
		synonym        Synonym
		expectedFields []string
		unexpectedFields []string
	}{
		{
			name: "multi-way synonym",
			synonym: Synonym{
				ID:       "fruit-synonyms",
				Synonyms: []string{"apple", "orange", "banana"},
			},
			expectedFields:   []string{"id", "synonyms"},
			unexpectedFields: []string{"root"},
		},
		{
			name: "one-way synonym with root",
			synonym: Synonym{
				ID:       "computer-synonyms",
				Root:     "computer",
				Synonyms: []string{"pc", "desktop", "workstation"},
			},
			expectedFields:   []string{"id", "synonyms", "root"},
			unexpectedFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.synonym)
			if err != nil {
				t.Fatalf("Failed to marshal Synonym: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			for _, field := range tt.expectedFields {
				if _, ok := result[field]; !ok {
					t.Errorf("Expected '%s' field in Synonym JSON", field)
				}
			}

			for _, field := range tt.unexpectedFields {
				if _, ok := result[field]; ok {
					t.Errorf("Did not expect '%s' field in Synonym JSON", field)
				}
			}
		})
	}
}

// =============================================================================
// Override API Payload Tests
// =============================================================================

func TestOverrideJSONSerialization(t *testing.T) {
	override := Override{
		ID: "featured-laptops",
		Rule: OverrideRule{
			Query: "laptop",
			Match: "exact",
			Tags:  []string{"featured", "sale"},
		},
		Includes: []OverrideInclude{
			{ID: "product-1", Position: 1},
			{ID: "product-2", Position: 2},
		},
		Excludes: []OverrideExclude{
			{ID: "product-99"},
		},
		FilterBy:            "category:electronics",
		SortBy:              "popularity:desc",
		ReplaceQuery:        "notebook computer",
		RemoveMatchedTokens: true,
		FilterCuratedHits:   true,
		EffectiveFromTs:     1704067200,
		EffectiveToTs:       1735689600,
		StopProcessing:      false,
		Metadata:            map[string]any{"campaign": "winter-sale"},
	}

	data, err := json.Marshal(override)
	if err != nil {
		t.Fatalf("Failed to marshal Override: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all expected field names match Typesense API
	expectedFields := map[string]string{
		"id":                    "id",
		"rule":                  "rule",
		"includes":              "includes",
		"excludes":              "excludes",
		"filter_by":             "filter_by",
		"sort_by":               "sort_by",
		"replace_query":         "replace_query",
		"remove_matched_tokens": "remove_matched_tokens",
		"filter_curated_hits":   "filter_curated_hits",
		"effective_from_ts":     "effective_from_ts",
		"effective_to_ts":       "effective_to_ts",
		"metadata":              "metadata",
	}

	for jsonField := range expectedFields {
		if _, ok := result[jsonField]; !ok {
			t.Errorf("Expected '%s' field in Override JSON", jsonField)
		}
	}

	// Verify rule structure
	rule, ok := result["rule"].(map[string]interface{})
	if !ok {
		t.Fatal("'rule' is not an object")
	}

	ruleFields := []string{"query", "match", "tags"}
	for _, field := range ruleFields {
		if _, ok := rule[field]; !ok {
			t.Errorf("Expected '%s' field in Override rule", field)
		}
	}

	// Verify includes structure
	includes := result["includes"].([]interface{})
	if len(includes) != 2 {
		t.Errorf("Expected 2 includes, got %d", len(includes))
	}

	firstInclude := includes[0].(map[string]interface{})
	if _, ok := firstInclude["id"]; !ok {
		t.Error("Expected 'id' field in include")
	}
	if _, ok := firstInclude["position"]; !ok {
		t.Error("Expected 'position' field in include")
	}
}

// =============================================================================
// StopwordsSet API Payload Tests
// =============================================================================

func TestStopwordsSetJSONSerialization(t *testing.T) {
	stopwords := StopwordsSet{
		ID:        "english-stopwords",
		Stopwords: []string{"the", "a", "an", "is", "are"},
		Locale:    "en",
	}

	data, err := json.Marshal(stopwords)
	if err != nil {
		t.Fatalf("Failed to marshal StopwordsSet: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	expectedFields := []string{"id", "stopwords", "locale"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in StopwordsSet JSON", field)
		}
	}

	// Verify stopwords is an array
	sw, ok := result["stopwords"].([]interface{})
	if !ok {
		t.Fatal("'stopwords' is not an array")
	}
	if len(sw) != 5 {
		t.Errorf("Expected 5 stopwords, got %d", len(sw))
	}
}

// =============================================================================
// APIKey API Payload Tests
// =============================================================================

func TestAPIKeyJSONSerialization(t *testing.T) {
	apiKey := APIKey{
		Description: "Search-only key",
		Actions:     []string{"documents:search"},
		Collections: []string{"products", "articles"},
		ExpiresAt:   1735689600,
	}

	data, err := json.Marshal(apiKey)
	if err != nil {
		t.Fatalf("Failed to marshal APIKey: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify required fields for creation
	expectedFields := []string{"description", "actions", "collections"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in APIKey JSON", field)
		}
	}

	// Verify expires_at when set
	if _, ok := result["expires_at"]; !ok {
		t.Error("Expected 'expires_at' field in APIKey JSON when set")
	}

	// ID and Value should not be serialized for creation (omitempty with zero values)
	if _, ok := result["id"]; ok {
		t.Error("Did not expect 'id' field in APIKey creation payload")
	}
}

// =============================================================================
// CurationItem (within CurationSet) API Payload Tests
// =============================================================================

func TestCurationItemJSONSerialization(t *testing.T) {
	item := CurationItem{
		ID: "promote-sale-items",
		Rule: OverrideRule{
			Query: "sale",
			Match: "contains",
		},
		Includes: []OverrideInclude{
			{ID: "item-1", Position: 1},
		},
		Excludes: []OverrideExclude{
			{ID: "item-99"},
		},
		FilterBy:            "in_stock:true",
		SortBy:              "discount:desc",
		ReplaceQuery:        "",
		RemoveMatchedTokens: false,
		FilterCuratedHits:   true,
		EffectiveFromTs:     1704067200,
		EffectiveToTs:       1735689600,
		StopProcessing:      true,
		Metadata:            map[string]any{"priority": "high"},
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal CurationItem: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify field names match API expectations
	expectedFields := []string{
		"id", "rule", "includes", "excludes", "filter_by", "sort_by",
		"filter_curated_hits", "effective_from_ts", "effective_to_ts",
		"stop_processing", "metadata",
	}

	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in CurationItem JSON", field)
		}
	}
}

// =============================================================================
// HTTP Mock Server Integration Tests
// =============================================================================
// These tests use a mock HTTP server to validate that actual API requests
// contain the correct JSON payloads.

func TestUpsertSynonymSetHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/synonym_sets/") {
			t.Errorf("Expected path starting with /synonym_sets/, got %s", r.URL.Path)
		}

		// Read and parse the request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		// Return a successful response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":  "test-set",
			"items": []interface{}{},
		})
	}))
	defer server.Close()

	// Parse the server URL to get host and port
	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	synonymSet := &SynonymSet{
		Name: "test-set",
		Synonyms: []SynonymItem{
			{
				ID:       "syn-1",
				Synonyms: []string{"word1", "word2"},
			},
		},
	}

	_, err := client.UpsertSynonymSet(context.Background(), synonymSet)
	if err != nil {
		t.Fatalf("UpsertSynonymSet failed: %v", err)
	}

	// Validate the payload sent to the server
	if _, ok := receivedPayload["items"]; !ok {
		t.Error("Request payload missing 'items' field - API requires 'items' not 'synonyms'")
	}
	if _, ok := receivedPayload["synonyms"]; ok {
		t.Error("Request payload should not have top-level 'synonyms' field")
	}
}

func TestUpsertCurationSetHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/curation_sets/") {
			t.Errorf("Expected path starting with /curation_sets/, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":  "test-curation",
			"items": []interface{}{},
		})
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	curationSet := &CurationSet{
		Name: "test-curation",
		Curations: []CurationItem{
			{
				ID: "cur-1",
				Rule: OverrideRule{
					Query: "test",
					Match: "exact",
				},
			},
		},
	}

	_, err := client.UpsertCurationSet(context.Background(), curationSet)
	if err != nil {
		t.Fatalf("UpsertCurationSet failed: %v", err)
	}

	// Validate the payload - API expects "items" field, not "curations"
	if _, ok := receivedPayload["items"]; !ok {
		t.Error("Request payload missing 'items' field - Typesense v30 API requires 'items' not 'curations'")
	}
	if _, ok := receivedPayload["name"]; !ok {
		t.Error("Request payload missing 'name' field")
	}
}

func TestCreateCollectionHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/collections" {
			t.Errorf("Expected path /collections, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(receivedPayload)
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	collection := &Collection{
		Name: "test-collection",
		Fields: []CollectionField{
			{Name: "title", Type: "string"},
			{Name: "price", Type: "float"},
		},
		DefaultSortingField: "price",
	}

	_, err := client.CreateCollection(context.Background(), collection)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	// Validate the payload
	if receivedPayload["name"] != "test-collection" {
		t.Errorf("Expected name 'test-collection', got %v", receivedPayload["name"])
	}
	if _, ok := receivedPayload["fields"]; !ok {
		t.Error("Request payload missing 'fields' field")
	}
	if receivedPayload["default_sorting_field"] != "price" {
		t.Errorf("Expected default_sorting_field 'price', got %v", receivedPayload["default_sorting_field"])
	}
}

func TestCreateSynonymHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		// v29 per-collection synonym path
		if !strings.Contains(r.URL.Path, "/collections/") || !strings.Contains(r.URL.Path, "/synonyms/") {
			t.Errorf("Expected path with /collections/.../synonyms/, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(receivedPayload)
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	synonym := &Synonym{
		ID:       "fruit-syn",
		Synonyms: []string{"apple", "orange", "banana"},
	}

	_, err := client.CreateSynonym(context.Background(), "products", synonym)
	if err != nil {
		t.Fatalf("CreateSynonym failed: %v", err)
	}

	// Validate the payload - for v29 per-collection synonyms
	if _, ok := receivedPayload["id"]; !ok {
		t.Error("Request payload missing 'id' field")
	}
	if _, ok := receivedPayload["synonyms"]; !ok {
		t.Error("Request payload missing 'synonyms' field")
	}
}

func TestCreateOverrideHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(receivedPayload)
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	override := &Override{
		ID: "test-override",
		Rule: OverrideRule{
			Query: "laptop",
			Match: "exact",
		},
		Includes: []OverrideInclude{
			{ID: "doc-1", Position: 1},
		},
	}

	_, err := client.CreateOverride(context.Background(), "products", override)
	if err != nil {
		t.Fatalf("CreateOverride failed: %v", err)
	}

	// Validate the payload
	if _, ok := receivedPayload["id"]; !ok {
		t.Error("Request payload missing 'id' field")
	}
	if _, ok := receivedPayload["rule"]; !ok {
		t.Error("Request payload missing 'rule' field")
	}
	if _, ok := receivedPayload["includes"]; !ok {
		t.Error("Request payload missing 'includes' field")
	}
}

func TestCreateStopwordsSetHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/stopwords/") {
			t.Errorf("Expected path starting with /stopwords/, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(receivedPayload)
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	stopwords := &StopwordsSet{
		ID:        "english",
		Stopwords: []string{"the", "a", "an"},
		Locale:    "en",
	}

	_, err := client.CreateStopwordsSet(context.Background(), stopwords)
	if err != nil {
		t.Fatalf("CreateStopwordsSet failed: %v", err)
	}

	// Validate the payload
	if _, ok := receivedPayload["id"]; !ok {
		t.Error("Request payload missing 'id' field")
	}
	if _, ok := receivedPayload["stopwords"]; !ok {
		t.Error("Request payload missing 'stopwords' field")
	}
	if _, ok := receivedPayload["locale"]; !ok {
		t.Error("Request payload missing 'locale' field")
	}
}

func TestCreateAPIKeyHTTPPayload(t *testing.T) {
	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/keys" {
			t.Errorf("Expected path /keys, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		response := map[string]interface{}{
			"id":          1,
			"value":       "generated-key-value",
			"description": receivedPayload["description"],
			"actions":     receivedPayload["actions"],
			"collections": receivedPayload["collections"],
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	apiKey := &APIKey{
		Description: "Search key",
		Actions:     []string{"documents:search"},
		Collections: []string{"*"},
	}

	_, err := client.CreateAPIKey(context.Background(), apiKey)
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	// Validate the payload
	if _, ok := receivedPayload["description"]; !ok {
		t.Error("Request payload missing 'description' field")
	}
	if _, ok := receivedPayload["actions"]; !ok {
		t.Error("Request payload missing 'actions' field")
	}
	if _, ok := receivedPayload["collections"]; !ok {
		t.Error("Request payload missing 'collections' field")
	}
}

// =============================================================================
// Round-Trip Serialization Tests
// =============================================================================
// These tests verify that structs can be serialized and deserialized without
// losing data, which is important for reading API responses.

func TestSynonymSetRoundTrip(t *testing.T) {
	original := SynonymSet{
		Name: "test-synonyms",
		Synonyms: []SynonymItem{
			{
				ID:       "syn-1",
				Root:     "laptop",
				Synonyms: []string{"notebook", "portable"},
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SynonymSet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, original.Name)
	}
	if len(decoded.Synonyms) != len(original.Synonyms) {
		t.Errorf("Synonyms length mismatch: got %d, want %d", len(decoded.Synonyms), len(original.Synonyms))
	}
	if decoded.Synonyms[0].ID != original.Synonyms[0].ID {
		t.Errorf("First item ID mismatch: got %s, want %s", decoded.Synonyms[0].ID, original.Synonyms[0].ID)
	}
}

func TestCollectionRoundTrip(t *testing.T) {
	indexTrue := true
	original := Collection{
		Name:                "test-collection",
		DefaultSortingField: "created_at",
		Fields: []CollectionField{
			{Name: "title", Type: "string", Facet: true, Index: &indexTrue},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Collection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, original.Name)
	}
	if decoded.DefaultSortingField != original.DefaultSortingField {
		t.Errorf("DefaultSortingField mismatch: got %s, want %s", decoded.DefaultSortingField, original.DefaultSortingField)
	}
	if len(decoded.Fields) != len(original.Fields) {
		t.Errorf("Fields length mismatch: got %d, want %d", len(decoded.Fields), len(original.Fields))
	}
}

// =============================================================================
// Analytics Rule API Payload Tests
// =============================================================================
// These tests verify that analytics rules are formatted correctly for both
// v30+ and pre-v30 Typesense API versions.

// TestUpsertAnalyticsRuleHTTPPayload_V30 validates that analytics rules sent to
// Typesense v30+ include the top-level 'collection' field. This test reproduces
// the issue where missing 'collection' field caused "Collection is required" error.
func TestUpsertAnalyticsRuleHTTPPayload_V30(t *testing.T) {
	var receivedPayload map[string]interface{}

	// Mock server that returns version 30.0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "30.0",
				"state":   1,
			})
			return
		}

		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/analytics/rules/") {
			t.Errorf("Expected path starting with /analytics/rules/, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":       "test-rule",
			"type":       "popular_queries",
			"collection": "products",
		})
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	rule := &AnalyticsRule{
		Name:       "test-rule",
		Type:       "popular_queries",
		Collection: "products",
		EventType:  "search",
		Params: map[string]any{
			"destination_collection": "product_queries",
			"limit":                  1000,
		},
	}

	_, err := client.UpsertAnalyticsRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("UpsertAnalyticsRule failed: %v", err)
	}

	// CRITICAL: Verify 'collection' field is present at top level
	// This is the exact issue that caused "Collection is required" error in v30+
	if _, ok := receivedPayload["collection"]; !ok {
		t.Error("Request payload missing 'collection' field - Typesense v30+ requires top-level 'collection'")
	}
	if receivedPayload["collection"] != "products" {
		t.Errorf("Expected collection 'products', got %v", receivedPayload["collection"])
	}

	// Verify other required fields
	if _, ok := receivedPayload["type"]; !ok {
		t.Error("Request payload missing 'type' field")
	}
	if _, ok := receivedPayload["event_type"]; !ok {
		t.Error("Request payload missing 'event_type' field")
	}
	if _, ok := receivedPayload["params"]; !ok {
		t.Error("Request payload missing 'params' field")
	}

	// Verify params uses flat format (destination_collection, not nested destination.collection)
	params, ok := receivedPayload["params"].(map[string]interface{})
	if !ok {
		t.Fatal("'params' is not an object")
	}
	if _, ok := params["destination_collection"]; !ok {
		t.Error("Expected 'destination_collection' in params for v30+ format")
	}
}

// TestUpsertAnalyticsRuleHTTPPayload_PreV30 validates that analytics rules sent to
// pre-v30 Typesense use the nested source.collections format.
func TestUpsertAnalyticsRuleHTTPPayload_PreV30(t *testing.T) {
	var receivedPayload map[string]interface{}

	// Mock server that returns version 29.0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"version": "29.0",
				"state":   1,
			})
			return
		}

		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("Failed to read request body: %v", err)
		}

		if err := json.Unmarshal(body, &receivedPayload); err != nil {
			t.Fatalf("Failed to parse request JSON: %v", err)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name": "test-rule",
			"type": "popular_queries",
		})
	}))
	defer server.Close()

	client := &ServerClient{
		httpClient: http.DefaultClient,
		apiKey:     "test-api-key",
		baseURL:    server.URL,
	}

	rule := &AnalyticsRule{
		Name:       "test-rule",
		Type:       "popular_queries",
		Collection: "products",
		EventType:  "search",
		Params: map[string]any{
			"destination_collection": "product_queries",
			"limit":                  1000,
		},
	}

	_, err := client.UpsertAnalyticsRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("UpsertAnalyticsRule failed: %v", err)
	}

	// Verify pre-v30 format: NO top-level collection field
	if _, ok := receivedPayload["collection"]; ok {
		t.Error("Pre-v30 format should NOT have top-level 'collection' field")
	}

	// Verify nested source.collections format
	params, ok := receivedPayload["params"].(map[string]interface{})
	if !ok {
		t.Fatal("'params' is not an object")
	}

	source, ok := params["source"].(map[string]interface{})
	if !ok {
		t.Fatal("Pre-v30 format should have 'source' object in params")
	}

	collections, ok := source["collections"].([]interface{})
	if !ok {
		t.Fatal("Pre-v30 format should have 'collections' array in source")
	}
	if len(collections) != 1 || collections[0] != "products" {
		t.Errorf("Expected collections ['products'], got %v", collections)
	}

	// Verify nested destination.collection format
	destination, ok := params["destination"].(map[string]interface{})
	if !ok {
		t.Fatal("Pre-v30 format should have 'destination' object in params")
	}
	if destination["collection"] != "product_queries" {
		t.Errorf("Expected destination.collection 'product_queries', got %v", destination["collection"])
	}
}

// TestAnalyticsRuleJSONSerialization validates that AnalyticsRule struct
// serializes correctly with the 'collection' field for v30+.
func TestAnalyticsRuleJSONSerialization(t *testing.T) {
	rule := AnalyticsRule{
		Name:       "test-analytics",
		Type:       "popular_queries",
		Collection: "products",
		EventType:  "search",
		Params: map[string]any{
			"destination_collection": "queries",
			"limit":                  500,
		},
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("Failed to marshal AnalyticsRule: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := []string{"name", "type", "collection", "event_type", "params"}
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected '%s' field in AnalyticsRule JSON", field)
		}
	}

	// Verify collection field value
	if result["collection"] != "products" {
		t.Errorf("Expected collection 'products', got %v", result["collection"])
	}
}

func TestOverrideRoundTrip(t *testing.T) {
	original := Override{
		ID: "test-override",
		Rule: OverrideRule{
			Query: "sale",
			Match: "contains",
		},
		Includes: []OverrideInclude{
			{ID: "doc-1", Position: 1},
		},
		FilterBy:          "active:true",
		StopProcessing:    true,
		Metadata:          map[string]any{"source": "test"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Override
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, original.ID)
	}
	if decoded.Rule.Query != original.Rule.Query {
		t.Errorf("Rule.Query mismatch: got %s, want %s", decoded.Rule.Query, original.Rule.Query)
	}
	if decoded.FilterBy != original.FilterBy {
		t.Errorf("FilterBy mismatch: got %s, want %s", decoded.FilterBy, original.FilterBy)
	}
	if decoded.StopProcessing != original.StopProcessing {
		t.Errorf("StopProcessing mismatch: got %v, want %v", decoded.StopProcessing, original.StopProcessing)
	}
}
