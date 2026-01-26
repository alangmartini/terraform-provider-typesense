package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServerClient handles communication with the Typesense Server API
type ServerClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// ServerInfo contains debug/version information from the Typesense server
type ServerInfo struct {
	State   int    `json:"state"`
	Version string `json:"version"`
}

// SynonymSet represents a Typesense synonym set (v30.0+)
type SynonymSet struct {
	Name     string        `json:"name"`
	Synonyms []SynonymItem `json:"synonyms,omitempty"`
}

// SynonymItem represents a synonym item within a synonym set (v30.0+)
type SynonymItem struct {
	ID       string   `json:"id"`
	Root     string   `json:"root,omitempty"`
	Synonyms []string `json:"synonyms"`
}

// CurationSet represents a Typesense curation set (v30.0+)
type CurationSet struct {
	Name       string         `json:"name"`
	Curations  []CurationItem `json:"curations,omitempty"`
}

// CurationItem represents a curation item within a curation set (v30.0+)
type CurationItem struct {
	ID                  string             `json:"id"`
	Rule                OverrideRule       `json:"rule"`
	Includes            []OverrideInclude  `json:"includes,omitempty"`
	Excludes            []OverrideExclude  `json:"excludes,omitempty"`
	FilterBy            string             `json:"filter_by,omitempty"`
	SortBy              string             `json:"sort_by,omitempty"`
	ReplaceQuery        string             `json:"replace_query,omitempty"`
	RemoveMatchedTokens bool               `json:"remove_matched_tokens,omitempty"`
	FilterCuratedHits   bool               `json:"filter_curated_hits,omitempty"`
	EffectiveFromTs     int64              `json:"effective_from_ts,omitempty"`
	EffectiveToTs       int64              `json:"effective_to_ts,omitempty"`
	StopProcessing      bool               `json:"stop_processing,omitempty"`
	Metadata            map[string]any     `json:"metadata,omitempty"`
}

// NewServerClient creates a new Server API client
func NewServerClient(host, apiKey string, port int, protocol string) *ServerClient {
	baseURL := fmt.Sprintf("%s://%s:%d", protocol, host, port)
	return &ServerClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

// Collection represents a Typesense collection
type Collection struct {
	Name                string            `json:"name"`
	Fields              []CollectionField `json:"fields"`
	DefaultSortingField string            `json:"default_sorting_field,omitempty"`
	TokenSeparators     []string          `json:"token_separators,omitempty"`
	SymbolsToIndex      []string          `json:"symbols_to_index,omitempty"`
	EnableNestedFields  bool              `json:"enable_nested_fields,omitempty"`
	NumDocuments        int64             `json:"num_documents,omitempty"`
	CreatedAt           int64             `json:"created_at,omitempty"`
}

// CollectionField represents a field in a collection schema
type CollectionField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Facet    bool   `json:"facet,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Index    *bool  `json:"index,omitempty"`
	Sort     bool   `json:"sort,omitempty"`
	Infix    bool   `json:"infix,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Drop     bool   `json:"drop,omitempty"`
}

// Synonym represents a Typesense synonym configuration
type Synonym struct {
	ID       string   `json:"id"`
	Root     string   `json:"root,omitempty"`
	Synonyms []string `json:"synonyms"`
}

// Override represents a Typesense curation/override rule
type Override struct {
	ID                  string              `json:"id"`
	Rule                OverrideRule        `json:"rule"`
	Includes            []OverrideInclude   `json:"includes,omitempty"`
	Excludes            []OverrideExclude   `json:"excludes,omitempty"`
	FilterBy            string              `json:"filter_by,omitempty"`
	SortBy              string              `json:"sort_by,omitempty"`
	ReplaceQuery        string              `json:"replace_query,omitempty"`
	RemoveMatchedTokens bool                `json:"remove_matched_tokens,omitempty"`
	FilterCuratedHits   bool                `json:"filter_curated_hits,omitempty"`
	EffectiveFromTs     int64               `json:"effective_from_ts,omitempty"`
	EffectiveToTs       int64               `json:"effective_to_ts,omitempty"`
	StopProcessing      bool                `json:"stop_processing,omitempty"`
	Metadata            map[string]any      `json:"metadata,omitempty"`
}

// OverrideRule defines when an override should apply
type OverrideRule struct {
	Query string `json:"query,omitempty"`
	Match string `json:"match,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// OverrideInclude specifies a document to include/pin
type OverrideInclude struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// OverrideExclude specifies a document to exclude
type OverrideExclude struct {
	ID string `json:"id"`
}

// StopwordsSet represents a Typesense stopwords set
type StopwordsSet struct {
	ID        string   `json:"id"`
	Stopwords []string `json:"stopwords"`
	Locale    string   `json:"locale,omitempty"`
}

// APIKey represents a Typesense API key
type APIKey struct {
	ID          int64    `json:"id,omitempty"`
	Value       string   `json:"value,omitempty"`
	Description string   `json:"description"`
	Actions     []string `json:"actions"`
	Collections []string `json:"collections"`
	ExpiresAt   int64    `json:"expires_at,omitempty"`
}

// CreateCollection creates a new collection
func (c *ServerClient) CreateCollection(ctx context.Context, collection *Collection) (*Collection, error) {
	body, err := json.Marshal(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/collections", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Collection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetCollection retrieves a collection by name
func (c *ServerClient) GetCollection(ctx context.Context, name string) (*Collection, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/collections/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get collection: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Collection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateCollection updates a collection's schema (add/drop fields)
func (c *ServerClient) UpdateCollection(ctx context.Context, name string, update *Collection) (*Collection, error) {
	body, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal collection update: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/collections/"+name, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update collection: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Collection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteCollection deletes a collection
func (c *ServerClient) DeleteCollection(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/collections/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete collection: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateSynonym creates or updates a synonym
func (c *ServerClient) CreateSynonym(ctx context.Context, collectionName string, synonym *Synonym) (*Synonym, error) {
	body, err := json.Marshal(synonym)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal synonym: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/synonyms/%s", c.baseURL, collectionName, synonym.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create synonym: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create synonym: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Synonym
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetSynonym retrieves a synonym by ID
func (c *ServerClient) GetSynonym(ctx context.Context, collectionName, synonymID string) (*Synonym, error) {
	url := fmt.Sprintf("%s/collections/%s/synonyms/%s", c.baseURL, collectionName, synonymID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get synonym: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get synonym: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Synonym
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteSynonym deletes a synonym
func (c *ServerClient) DeleteSynonym(ctx context.Context, collectionName, synonymID string) error {
	url := fmt.Sprintf("%s/collections/%s/synonyms/%s", c.baseURL, collectionName, synonymID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete synonym: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete synonym: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateOverride creates or updates an override/curation rule
func (c *ServerClient) CreateOverride(ctx context.Context, collectionName string, override *Override) (*Override, error) {
	body, err := json.Marshal(override)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal override: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/overrides/%s", c.baseURL, collectionName, override.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create override: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create override: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Override
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetOverride retrieves an override by ID
func (c *ServerClient) GetOverride(ctx context.Context, collectionName, overrideID string) (*Override, error) {
	url := fmt.Sprintf("%s/collections/%s/overrides/%s", c.baseURL, collectionName, overrideID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get override: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get override: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Override
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteOverride deletes an override
func (c *ServerClient) DeleteOverride(ctx context.Context, collectionName, overrideID string) error {
	url := fmt.Sprintf("%s/collections/%s/overrides/%s", c.baseURL, collectionName, overrideID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete override: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete override: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateStopwordsSet creates or updates a stopwords set
func (c *ServerClient) CreateStopwordsSet(ctx context.Context, stopwords *StopwordsSet) (*StopwordsSet, error) {
	body, err := json.Marshal(stopwords)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stopwords: %w", err)
	}

	url := fmt.Sprintf("%s/stopwords/%s", c.baseURL, stopwords.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create stopwords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create stopwords: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result StopwordsSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetStopwordsSet retrieves a stopwords set by ID
func (c *ServerClient) GetStopwordsSet(ctx context.Context, id string) (*StopwordsSet, error) {
	url := fmt.Sprintf("%s/stopwords/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stopwords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get stopwords: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// The API returns {"stopwords": {...}} wrapper
	var wrapper struct {
		Stopwords StopwordsSet `json:"stopwords"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.Stopwords, nil
}

// DeleteStopwordsSet deletes a stopwords set
func (c *ServerClient) DeleteStopwordsSet(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/stopwords/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete stopwords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete stopwords: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// CreateAPIKey creates a new API key
func (c *ServerClient) CreateAPIKey(ctx context.Context, key *APIKey) (*APIKey, error) {
	body, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API key: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/keys", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create API key: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAPIKey retrieves an API key by ID
func (c *ServerClient) GetAPIKey(ctx context.Context, id int64) (*APIKey, error) {
	url := fmt.Sprintf("%s/keys/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get API key: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result APIKey
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteAPIKey deletes an API key
func (c *ServerClient) DeleteAPIKey(ctx context.Context, id int64) error {
	url := fmt.Sprintf("%s/keys/%d", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete API key: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *ServerClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
}

// GetServerInfo retrieves debug/version information from the server
func (c *ServerClient) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/debug", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get server info: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ServerInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListSynonymSets retrieves all synonym sets (Typesense v30.0+)
func (c *ServerClient) ListSynonymSets(ctx context.Context) ([]SynonymSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/synonym_sets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list synonym sets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Endpoint doesn't exist, likely older Typesense version
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list synonym sets: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []SynonymSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ListCurationSets retrieves all curation sets (Typesense v30.0+)
func (c *ServerClient) ListCurationSets(ctx context.Context) ([]CurationSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/curation_sets", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list curation sets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Endpoint doesn't exist, likely older Typesense version
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list curation sets: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []CurationSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ListCollections retrieves all collections
func (c *ServerClient) ListCollections(ctx context.Context) ([]Collection, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/collections", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list collections: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []Collection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// ListSynonyms retrieves all synonyms for a collection (Typesense v29 and earlier)
// For Typesense v30+, this endpoint doesn't exist - use ListSynonymSets instead.
// Returns an empty list if the endpoint doesn't exist (404).
func (c *ServerClient) ListSynonyms(ctx context.Context, collectionName string) ([]Synonym, error) {
	url := fmt.Sprintf("%s/collections/%s/synonyms", c.baseURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list synonyms: %w", err)
	}
	defer resp.Body.Close()

	// In Typesense 30.0+, the per-collection synonyms endpoint no longer exists
	// Return empty list instead of error to allow graceful fallback
	if resp.StatusCode == http.StatusNotFound {
		return []Synonym{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list synonyms: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// The API returns {"synonyms": [...]}
	var wrapper struct {
		Synonyms []Synonym `json:"synonyms"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return wrapper.Synonyms, nil
}

// ListOverrides retrieves all overrides for a collection (Typesense v29 and earlier)
// For Typesense v30+, this endpoint doesn't exist - use ListCurationSets instead.
// Returns an empty list if the endpoint doesn't exist (404).
func (c *ServerClient) ListOverrides(ctx context.Context, collectionName string) ([]Override, error) {
	url := fmt.Sprintf("%s/collections/%s/overrides", c.baseURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list overrides: %w", err)
	}
	defer resp.Body.Close()

	// In Typesense 30.0+, the per-collection overrides endpoint no longer exists
	// Return empty list instead of error to allow graceful fallback
	if resp.StatusCode == http.StatusNotFound {
		return []Override{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list overrides: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// The API returns {"overrides": [...]}
	var wrapper struct {
		Overrides []Override `json:"overrides"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return wrapper.Overrides, nil
}

// ListStopwordsSets retrieves all stopwords sets
func (c *ServerClient) ListStopwordsSets(ctx context.Context) ([]StopwordsSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/stopwords", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list stopwords: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list stopwords: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// The API returns {"stopwords": [...]}
	var wrapper struct {
		Stopwords []StopwordsSet `json:"stopwords"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return wrapper.Stopwords, nil
}

