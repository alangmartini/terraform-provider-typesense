//go:build e2e

package chinooktest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestMigrateV30 stands up two v30 clusters, applies chinook to source,
// seeds a handful of tracks documents, runs `generate --include-data`
// against source, then `migrate --include-documents` against target, and
// asserts schema fingerprints and document counts match across the two
// clusters.
func TestMigrateV30(t *testing.T) {
	source := StartCluster(t, "30.1")
	target := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)

	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	if err := tf.Apply(chinookVars(source, m.Vars)); err != nil {
		t.Fatalf("apply chinook to source: %v", err)
	}

	if err := seedTracks(source, sampleTracks); err != nil {
		t.Fatalf("seed source tracks: %v", err)
	}

	exportDir := t.TempDir()
	if err := runGenerate(t, source, exportDir); err != nil {
		t.Fatalf("generate --include-data: %v", err)
	}

	if err := runMigrate(t, target, exportDir); err != nil {
		t.Fatalf("migrate to target: %v", err)
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

	srcFp, err := schemaFingerprint(source, "tracks")
	if err != nil {
		t.Fatalf("source fingerprint: %v", err)
	}
	tgtFp, err := schemaFingerprint(target, "tracks")
	if err != nil {
		t.Fatalf("target fingerprint: %v", err)
	}
	if srcFp != tgtFp {
		t.Errorf("tracks schema fingerprint differs\nsource: %s\ntarget: %s", srcFp, tgtFp)
	}
}

func runMigrate(t *testing.T, c *Cluster, sourceDir string) error {
	t.Helper()
	bin := filepath.Join(providerBinaryDir, "terraform-provider-typesense")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command(bin, "migrate",
		"--source-dir", sourceDir,
		"--target-host", c.Host,
		"--target-port", fmt.Sprintf("%d", c.Port),
		"--target-protocol", "http",
		"--target-api-key", c.APIKey,
		"--include-documents",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("migrate: %w\n%s", err, out)
	}
	return nil
}

// sampleTracks is a small synthetic dataset with every required tracks
// field populated. Three rows is plenty to exercise the document export
// and import paths without inflating wall-clock.
var sampleTracks = []map[string]any{
	{
		"id":               "1",
		"name":             "Rock You Like A Hurricane",
		"milliseconds":     257000,
		"unit_price":       0.99,
		"album_id":         "1",
		"album_title":      "Crazy World",
		"artist_id":        "1",
		"artist_name":      "Scorpions",
		"genre_id":         "1",
		"genre_name":       "Rock",
		"media_type_id":    "1",
		"media_type_name":  "MPEG audio file",
		"popularity_score": 87,
	},
	{
		"id":               "2",
		"name":             "So What",
		"milliseconds":     565000,
		"unit_price":       0.99,
		"album_id":         "2",
		"album_title":      "Kind of Blue",
		"artist_id":        "2",
		"artist_name":      "Miles Davis",
		"genre_id":         "2",
		"genre_name":       "Jazz",
		"media_type_id":    "1",
		"media_type_name":  "MPEG audio file",
		"popularity_score": 92,
	},
	{
		"id":               "3",
		"name":             "Take Five",
		"milliseconds":     324000,
		"unit_price":       1.29,
		"album_id":         "3",
		"album_title":      "Time Out",
		"artist_id":        "3",
		"artist_name":      "Dave Brubeck",
		"genre_id":         "2",
		"genre_name":       "Jazz",
		"media_type_id":    "1",
		"media_type_name":  "MPEG audio file",
		"popularity_score": 88,
	},
}

func seedTracks(c *Cluster, docs []map[string]any) error {
	var jsonl bytes.Buffer
	for _, d := range docs {
		b, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("marshal doc: %w", err)
		}
		jsonl.Write(b)
		jsonl.WriteByte('\n')
	}

	url := fmt.Sprintf("%s/collections/tracks/documents/import?action=upsert", c.BaseURL)
	req, err := http.NewRequest(http.MethodPost, url, &jsonl)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("import status %d", resp.StatusCode)
	}
	return nil
}

func collectionNames(colls []client.Collection) []string {
	names := make([]string, len(colls))
	for i, c := range colls {
		names[i] = c.Name
	}
	sort.Strings(names)
	return names
}

// schemaFingerprint reads the named collection from the cluster and
// returns a deterministic string capturing the field names and types,
// safe to compare across two clusters via string equality.
func schemaFingerprint(c *Cluster, name string) (string, error) {
	url := fmt.Sprintf("%s/collections/%s", c.BaseURL, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get collection status %d", resp.StatusCode)
	}

	var body struct {
		Fields []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	parts := make([]string, len(body.Fields))
	for i, f := range body.Fields {
		parts[i] = f.Name + ":" + f.Type
	}
	sort.Strings(parts)
	return strings.Join(parts, ","), nil
}

func collectionDocCount(c *Cluster, name string) (int64, error) {
	url := fmt.Sprintf("%s/collections/%s", c.BaseURL, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	var body struct {
		NumDocuments int64 `json:"num_documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	return body.NumDocuments, nil
}
