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
