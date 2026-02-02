package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// exportDocuments exports both schema and documents for a collection
func (g *Generator) exportDocuments(ctx context.Context, collectionName string) error {
	// Create data directory
	dataDir := filepath.Join(g.config.OutputDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Export schema
	if err := g.exportSchema(ctx, collectionName, dataDir); err != nil {
		return fmt.Errorf("failed to export schema: %w", err)
	}

	// Export documents
	if err := g.exportDocumentsToFile(ctx, collectionName, dataDir); err != nil {
		return fmt.Errorf("failed to export documents: %w", err)
	}

	return nil
}

// exportSchema saves the collection schema as JSON
func (g *Generator) exportSchema(ctx context.Context, collectionName string, dataDir string) error {
	collection, err := g.serverClient.GetCollection(ctx, collectionName)
	if err != nil {
		return err
	}

	// Remove computed fields that shouldn't be used during creation
	exportSchema := &client.Collection{
		Name:                collection.Name,
		Fields:              collection.Fields,
		DefaultSortingField: collection.DefaultSortingField,
		TokenSeparators:     collection.TokenSeparators,
		SymbolsToIndex:      collection.SymbolsToIndex,
		EnableNestedFields:  collection.EnableNestedFields,
	}

	schemaPath := filepath.Join(dataDir, collectionName+".schema.json")
	schemaFile, err := os.Create(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to create schema file: %w", err)
	}
	defer schemaFile.Close()

	encoder := json.NewEncoder(schemaFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportSchema); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	return nil
}

// exportDocumentsToFile streams documents from a collection to a JSONL file
func (g *Generator) exportDocumentsToFile(ctx context.Context, collectionName string, dataDir string) error {
	// Build export URL
	url := fmt.Sprintf("%s://%s:%d/collections/%s/documents/export",
		g.config.Protocol, g.config.Host, g.config.Port, collectionName)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", g.config.APIKey)

	// Execute request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to export documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("export failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Create output file
	outputPath := filepath.Join(dataDir, collectionName+".jsonl")
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Stream response directly to file (memory efficient)
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write documents: %w", err)
	}

	fmt.Printf("  Exported %s: %d bytes\n", collectionName, written)
	return nil
}

// exportSynonyms exports all synonyms for a collection to a JSON file
func (g *Generator) exportSynonyms(ctx context.Context, collectionName string, dataDir string) error {
	synonyms, err := g.serverClient.ListSynonyms(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to list synonyms: %w", err)
	}

	if len(synonyms) == 0 {
		return nil
	}

	synonymsPath := filepath.Join(dataDir, collectionName+".synonyms.json")
	synonymsFile, err := os.Create(synonymsPath)
	if err != nil {
		return fmt.Errorf("failed to create synonyms file: %w", err)
	}
	defer synonymsFile.Close()

	encoder := json.NewEncoder(synonymsFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(synonyms); err != nil {
		return fmt.Errorf("failed to write synonyms: %w", err)
	}

	fmt.Printf("  Exported %d synonyms for %s\n", len(synonyms), collectionName)
	return nil
}

// exportOverrides exports all overrides for a collection to a JSON file
func (g *Generator) exportOverrides(ctx context.Context, collectionName string, dataDir string) error {
	overrides, err := g.serverClient.ListOverrides(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to list overrides: %w", err)
	}

	if len(overrides) == 0 {
		return nil
	}

	overridesPath := filepath.Join(dataDir, collectionName+".overrides.json")
	overridesFile, err := os.Create(overridesPath)
	if err != nil {
		return fmt.Errorf("failed to create overrides file: %w", err)
	}
	defer overridesFile.Close()

	encoder := json.NewEncoder(overridesFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(overrides); err != nil {
		return fmt.Errorf("failed to write overrides: %w", err)
	}

	fmt.Printf("  Exported %d overrides for %s\n", len(overrides), collectionName)
	return nil
}

// exportStopwordsSets exports all stopwords sets to a JSON file
func (g *Generator) exportStopwordsSets(ctx context.Context, dataDir string) error {
	stopwordsSets, err := g.serverClient.ListStopwordsSets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list stopwords sets: %w", err)
	}

	if len(stopwordsSets) == 0 {
		return nil
	}

	stopwordsPath := filepath.Join(dataDir, "_stopwords.json")
	stopwordsFile, err := os.Create(stopwordsPath)
	if err != nil {
		return fmt.Errorf("failed to create stopwords file: %w", err)
	}
	defer stopwordsFile.Close()

	encoder := json.NewEncoder(stopwordsFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stopwordsSets); err != nil {
		return fmt.Errorf("failed to write stopwords: %w", err)
	}

	fmt.Printf("  Exported %d stopwords sets\n", len(stopwordsSets))
	return nil
}
