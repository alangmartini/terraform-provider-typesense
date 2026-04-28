//go:build e2e

package chinooktest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMigrateV23ToV30 stands up a v0.23.1 source cluster and a v30.1 target,
// seeds the source via direct HTTP (no terraform apply because the provider
// targets v26+), runs `generate --include-data` against the v23 source, then
// `migrate --include-documents` against the target, and asserts collections
// and documents come across intact.
//
// This is the only chinook test that exercises the "skip endpoints absent
// on this server version" code path. v23 lacks /stopwords, /analytics/rules,
// /stemming/dictionaries, /nl_search_models, /conversation_models,
// /synonym_sets, and /curation_sets. The generator must skip them with a
// warning rather than aborting.
func TestMigrateV23ToV30(t *testing.T) {
	source := StartCluster(t, "0.23.1")
	target := StartCluster(t, "30.1")

	if err := seedV23Source(source); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	exportDir := t.TempDir()
	if out, err := runGenerateCapture(t, source, exportDir); err != nil {
		t.Fatalf("generate against v23 source: %v\noutput:\n%s", err, out)
	}

	for _, name := range []string{
		"stopwords.tf",
		"analytics.tf",
		"stemming.tf",
		"nl_search_models.tf",
		"conversation_models.tf",
	} {
		if _, err := os.Stat(filepath.Join(exportDir, name)); err == nil {
			t.Errorf("generate emitted %s for v23 source, want absent", name)
		}
	}

	for _, name := range []string{
		"collections.tf",
		"aliases.tf",
		"api_keys.tf",
	} {
		if _, err := os.Stat(filepath.Join(exportDir, name)); err != nil {
			t.Errorf("generate did not emit %s for v23 source: %v", name, err)
		}
	}

	if err := runMigrate(t, target, exportDir); err != nil {
		t.Fatalf("migrate v23 export to v30 target: %v", err)
	}

	ctx := context.Background()
	srcColls, err := source.Client().ListCollections(ctx)
	if err != nil {
		t.Fatalf("list source collections: %v", err)
	}
	tgtColls, err := target.Client().ListCollections(ctx)
	if err != nil {
		t.Fatalf("list target collections: %v", err)
	}

	srcNames := collectionNames(srcColls)
	tgtNames := collectionNames(tgtColls)
	if !equalStringSlices(srcNames, tgtNames) {
		t.Errorf("collection names differ\nsource: %v\ntarget: %v", srcNames, tgtNames)
	}

	for _, name := range srcNames {
		srcCount, err := collectionDocCount(source, name)
		if err != nil {
			t.Errorf("source num_documents %s: %v", name, err)
			continue
		}
		tgtCount, err := collectionDocCount(target, name)
		if err != nil {
			t.Errorf("target num_documents %s: %v", name, err)
			continue
		}
		if srcCount != tgtCount {
			t.Errorf("collection %s: source=%d, target=%d", name, srcCount, tgtCount)
		}
	}

	srcFp, err := schemaFingerprint(source, "tracks_v23")
	if err != nil {
		t.Fatalf("source fingerprint: %v", err)
	}
	tgtFp, err := schemaFingerprint(target, "tracks_v23")
	if err != nil {
		t.Fatalf("target fingerprint: %v", err)
	}
	if srcFp != tgtFp {
		t.Errorf("tracks_v23 schema fingerprint differs\nsource: %s\ntarget: %s", srcFp, tgtFp)
	}
}

// seedV23Source creates a single collection with three documents directly via
// the source API. We bypass terraform here because the provider's minimum
// supported version is v26 and chinook resources are v27+.
func seedV23Source(c *Cluster) error {
	schema := map[string]any{
		"name": "tracks_v23",
		"fields": []map[string]any{
			{"name": "title", "type": "string"},
			{"name": "artist", "type": "string", "facet": true},
			{"name": "duration_ms", "type": "int32"},
		},
		"default_sorting_field": "duration_ms",
	}
	if err := postJSON(c, "/collections", schema, http.StatusCreated); err != nil {
		return fmt.Errorf("create tracks_v23 collection: %w", err)
	}

	docs := []map[string]any{
		{"id": "1", "title": "Smoke on the Water", "artist": "Deep Purple", "duration_ms": 340000},
		{"id": "2", "title": "Stairway to Heaven", "artist": "Led Zeppelin", "duration_ms": 482000},
		{"id": "3", "title": "Hotel California", "artist": "Eagles", "duration_ms": 391000},
	}
	var jsonl bytes.Buffer
	for _, d := range docs {
		b, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("marshal doc: %w", err)
		}
		jsonl.Write(b)
		jsonl.WriteByte('\n')
	}
	url := c.BaseURL + "/collections/tracks_v23/documents/import?action=upsert"
	req, err := http.NewRequest(http.MethodPost, url, &jsonl)
	if err != nil {
		return fmt.Errorf("new import request: %w", err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	req.Header.Set("Content-Type", "application/x-ndjson")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do import: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := readBody(resp)
		return fmt.Errorf("import status %d: %s", resp.StatusCode, body)
	}

	alias := map[string]any{"collection_name": "tracks_v23"}
	if err := putJSON(c, "/aliases/tracks_alias", alias, http.StatusOK); err != nil {
		return fmt.Errorf("create alias: %w", err)
	}

	apiKey := map[string]any{
		"description": "v23 search-only key",
		"actions":     []string{"documents:search"},
		"collections": []string{"tracks_v23"},
	}
	if err := postJSON(c, "/keys", apiKey, http.StatusCreated); err != nil {
		return fmt.Errorf("create api key: %w", err)
	}

	return nil
}

func postJSON(c *Cluster, path string, body any, wantStatus int) error {
	return doJSON(c, http.MethodPost, path, body, wantStatus)
}

func putJSON(c *Cluster, path string, body any, wantStatus int) error {
	return doJSON(c, http.MethodPut, path, body, wantStatus)
}

func doJSON(c *Cluster, method, path string, body any, wantStatus int) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		respBody, _ := readBody(resp)
		return fmt.Errorf("%s %s: status %d, want %d, body: %s", method, path, resp.StatusCode, wantStatus, respBody)
	}
	return nil
}

func readBody(resp *http.Response) (string, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(resp.Body)
	return buf.String(), err
}

// runGenerateCapture is like runGenerate but returns the combined stdout+stderr
// so callers can assert on warnings emitted to stderr.
func runGenerateCapture(t *testing.T, c *Cluster, outputDir string) (string, error) {
	t.Helper()
	bin := filepath.Join(providerBinaryDir, "terraform-provider-typesense")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command(bin, "generate",
		"--host", c.Host,
		"--port", fmt.Sprintf("%d", c.Port),
		"--protocol", "http",
		"--api-key", c.APIKey,
		"--output", outputDir,
		"--include-data",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("generate: %w", err)
	}
	return string(out), nil
}
