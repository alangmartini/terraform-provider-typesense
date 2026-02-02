package migrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// Config holds the configuration for the migrator
type Config struct {
	SourceDir      string
	TargetHost     string
	TargetPort     int
	TargetProtocol string
	TargetAPIKey   string
}

// Migrator handles importing data to a target Typesense cluster
type Migrator struct {
	config       *Config
	targetClient *client.ServerClient
	httpClient   *http.Client
	baseURL      string
}

// New creates a new Migrator with the given configuration
func New(cfg *Config) *Migrator {
	return &Migrator{
		config:       cfg,
		targetClient: client.NewServerClient(cfg.TargetHost, cfg.TargetAPIKey, cfg.TargetPort, cfg.TargetProtocol),
		httpClient: &http.Client{
			Timeout: 0, // No timeout for large imports
		},
		baseURL: fmt.Sprintf("%s://%s:%d", cfg.TargetProtocol, cfg.TargetHost, cfg.TargetPort),
	}
}

// Migrate imports all collections and documents from the source directory
func (m *Migrator) Migrate(ctx context.Context) error {
	dataDir := filepath.Join(m.config.SourceDir, "data")

	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return fmt.Errorf("data directory not found: %s (did you run generate with --include-data?)", dataDir)
	}

	// Find all schema files
	schemaFiles, err := filepath.Glob(filepath.Join(dataDir, "*.schema.json"))
	if err != nil {
		return fmt.Errorf("failed to find schema files: %w", err)
	}

	if len(schemaFiles) == 0 {
		fmt.Println("No collections found to migrate")
		return nil
	}

	fmt.Printf("Found %d collection(s) to migrate\n\n", len(schemaFiles))

	// Process each collection
	for _, schemaFile := range schemaFiles {
		collectionName := strings.TrimSuffix(filepath.Base(schemaFile), ".schema.json")

		fmt.Printf("Migrating collection: %s\n", collectionName)

		// Create collection
		if err := m.createCollection(ctx, schemaFile); err != nil {
			return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
		}

		// Import documents
		documentsFile := filepath.Join(dataDir, collectionName+".jsonl")
		if _, err := os.Stat(documentsFile); err == nil {
			if err := m.importDocuments(ctx, collectionName, documentsFile); err != nil {
				return fmt.Errorf("failed to import documents for %s: %w", collectionName, err)
			}
		} else {
			fmt.Printf("  No documents file found, skipping data import\n")
		}

		// Import synonyms if file exists
		synonymsFile := filepath.Join(dataDir, collectionName+".synonyms.json")
		if _, err := os.Stat(synonymsFile); err == nil {
			if err := m.importSynonyms(ctx, collectionName, synonymsFile); err != nil {
				return fmt.Errorf("failed to import synonyms for %s: %w", collectionName, err)
			}
		}

		// Import overrides if file exists
		overridesFile := filepath.Join(dataDir, collectionName+".overrides.json")
		if _, err := os.Stat(overridesFile); err == nil {
			if err := m.importOverrides(ctx, collectionName, overridesFile); err != nil {
				return fmt.Errorf("failed to import overrides for %s: %w", collectionName, err)
			}
		}
	}

	// Import global stopwords sets if file exists
	stopwordsFile := filepath.Join(dataDir, "_stopwords.json")
	if _, err := os.Stat(stopwordsFile); err == nil {
		fmt.Println()
		if err := m.importStopwordsSets(ctx, stopwordsFile); err != nil {
			return fmt.Errorf("failed to import stopwords sets: %w", err)
		}
	}

	return nil
}

// createCollection creates a collection on the target from a schema file
func (m *Migrator) createCollection(ctx context.Context, schemaFile string) error {
	// Read schema file
	data, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema client.Collection
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Check if collection already exists
	existing, err := m.targetClient.GetCollection(ctx, schema.Name)
	if err != nil {
		return fmt.Errorf("failed to check existing collection: %w", err)
	}

	if existing != nil {
		fmt.Printf("  Collection already exists, skipping creation\n")
		return nil
	}

	// Create collection
	_, err = m.targetClient.CreateCollection(ctx, &schema)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	fmt.Printf("  Created collection\n")
	return nil
}

// importDocuments streams documents from a JSONL file to the target cluster
func (m *Migrator) importDocuments(ctx context.Context, collectionName string, documentsFile string) error {
	// Get file info for size
	fileInfo, err := os.Stat(documentsFile)
	if err != nil {
		return fmt.Errorf("failed to stat documents file: %w", err)
	}

	if fileInfo.Size() == 0 {
		fmt.Printf("  No documents to import (empty file)\n")
		return nil
	}

	// Count lines for progress
	lineCount, err := countLines(documentsFile)
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}

	fmt.Printf("  Importing %d documents (%d bytes)...\n", lineCount, fileInfo.Size())

	// Open file for streaming
	file, err := os.Open(documentsFile)
	if err != nil {
		return fmt.Errorf("failed to open documents file: %w", err)
	}
	defer file.Close()

	// Create import request with streaming body
	url := fmt.Sprintf("%s/collections/%s/documents/import?action=upsert", m.baseURL, collectionName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, file)
	if err != nil {
		return fmt.Errorf("failed to create import request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-TYPESENSE-API-KEY", m.config.TargetAPIKey)
	req.ContentLength = fileInfo.Size()

	// Execute request
	startTime := time.Now()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("import request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Process response line by line to count successes/failures
	success, failed := m.processImportResponse(resp.Body)
	elapsed := time.Since(startTime)

	fmt.Printf("  Imported: %d success, %d failed (%.2fs)\n", success, failed, elapsed.Seconds())

	if failed > 0 {
		fmt.Printf("  Warning: %d documents failed to import\n", failed)
	}

	return nil
}

// processImportResponse reads the import response and counts successes/failures
func (m *Migrator) processImportResponse(body io.Reader) (success, failed int) {
	scanner := bufio.NewScanner(body)
	// Increase buffer size for large documents
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"success":true`) {
			success++
		} else if strings.Contains(line, `"success":false`) {
			failed++
		}
	}

	return success, failed
}

// countLines counts the number of lines in a file
func countLines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	// Increase buffer for large lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}

// importSynonyms imports synonyms for a collection from a JSON file
func (m *Migrator) importSynonyms(ctx context.Context, collectionName string, synonymsFile string) error {
	data, err := os.ReadFile(synonymsFile)
	if err != nil {
		return fmt.Errorf("failed to read synonyms file: %w", err)
	}

	var synonyms []client.Synonym
	if err := json.Unmarshal(data, &synonyms); err != nil {
		return fmt.Errorf("failed to parse synonyms: %w", err)
	}

	if len(synonyms) == 0 {
		return nil
	}

	for _, syn := range synonyms {
		_, err := m.targetClient.CreateSynonym(ctx, collectionName, &syn)
		if err != nil {
			return fmt.Errorf("failed to create synonym %s: %w", syn.ID, err)
		}
	}

	fmt.Printf("  Imported %d synonyms\n", len(synonyms))
	return nil
}

// importOverrides imports overrides for a collection from a JSON file
func (m *Migrator) importOverrides(ctx context.Context, collectionName string, overridesFile string) error {
	data, err := os.ReadFile(overridesFile)
	if err != nil {
		return fmt.Errorf("failed to read overrides file: %w", err)
	}

	var overrides []client.Override
	if err := json.Unmarshal(data, &overrides); err != nil {
		return fmt.Errorf("failed to parse overrides: %w", err)
	}

	if len(overrides) == 0 {
		return nil
	}

	for _, ovr := range overrides {
		_, err := m.targetClient.CreateOverride(ctx, collectionName, &ovr)
		if err != nil {
			return fmt.Errorf("failed to create override %s: %w", ovr.ID, err)
		}
	}

	fmt.Printf("  Imported %d overrides\n", len(overrides))
	return nil
}

// importStopwordsSets imports stopwords sets from a JSON file
func (m *Migrator) importStopwordsSets(ctx context.Context, stopwordsFile string) error {
	data, err := os.ReadFile(stopwordsFile)
	if err != nil {
		return fmt.Errorf("failed to read stopwords file: %w", err)
	}

	var stopwordsSets []client.StopwordsSet
	if err := json.Unmarshal(data, &stopwordsSets); err != nil {
		return fmt.Errorf("failed to parse stopwords: %w", err)
	}

	if len(stopwordsSets) == 0 {
		return nil
	}

	for _, sw := range stopwordsSets {
		_, err := m.targetClient.CreateStopwordsSet(ctx, &sw)
		if err != nil {
			return fmt.Errorf("failed to create stopwords set %s: %w", sw.ID, err)
		}
	}

	fmt.Printf("Imported %d stopwords sets\n", len(stopwordsSets))
	return nil
}
