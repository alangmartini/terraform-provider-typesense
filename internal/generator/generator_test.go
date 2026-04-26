package generator

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
	"github.com/alanm/terraform-provider-typesense/internal/version"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func TestClusterMatchesHost(t *testing.T) {
	tests := []struct {
		name    string
		cluster client.Cluster
		host    string
		want    bool
	}{
		{
			name: "matches load balanced hostname",
			cluster: client.Cluster{
				Hostnames: client.ClusterHostnames{
					SearchDeliveryNetwork: "abc123.a1.typesense.net",
					LoadBalanced:          "abc123.a1.typesense.net",
					Nodes:                 []string{"abc123-1.a1.typesense.net", "abc123-2.a1.typesense.net"},
				},
			},
			host: "abc123.a1.typesense.net",
			want: true,
		},
		{
			name: "matches first node hostname",
			cluster: client.Cluster{
				Hostnames: client.ClusterHostnames{
					SearchDeliveryNetwork: "abc123.a1.typesense.net",
					LoadBalanced:          "abc123.a1.typesense.net",
					Nodes:                 []string{"abc123-1.a1.typesense.net", "abc123-2.a1.typesense.net"},
				},
			},
			host: "abc123-1.a1.typesense.net",
			want: true,
		},
		{
			name: "matches second node hostname",
			cluster: client.Cluster{
				Hostnames: client.ClusterHostnames{
					SearchDeliveryNetwork: "abc123.a1.typesense.net",
					LoadBalanced:          "abc123.a1.typesense.net",
					Nodes:                 []string{"abc123-1.a1.typesense.net", "abc123-2.a1.typesense.net"},
				},
			},
			host: "abc123-2.a1.typesense.net",
			want: true,
		},
		{
			name: "does not match different cluster",
			cluster: client.Cluster{
				Hostnames: client.ClusterHostnames{
					SearchDeliveryNetwork: "abc123.a1.typesense.net",
					LoadBalanced:          "abc123.a1.typesense.net",
					Nodes:                 []string{"abc123-1.a1.typesense.net"},
				},
			},
			host: "xyz789-1.a1.typesense.net",
			want: false,
		},
		{
			name: "empty hostnames",
			cluster: client.Cluster{
				Hostnames: client.ClusterHostnames{},
			},
			host: "abc123-1.a1.typesense.net",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clusterMatchesHost(&tt.cluster, tt.host)
			if got != tt.want {
				t.Errorf("clusterMatchesHost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newGeneratorForTestServer(t *testing.T, handler http.HandlerFunc) (*Generator, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	serverURL := strings.TrimPrefix(server.URL, "http://")
	host, portStr, err := net.SplitHostPort(serverURL)
	if err != nil {
		server.Close()
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		server.Close()
		t.Fatalf("failed to parse test server port: %v", err)
	}

	g := New(&Config{
		Host:     host,
		Port:     port,
		Protocol: "http",
		APIKey:   "test-key",
	})

	return g, server.Close
}

func TestGenerateSynonymSetsV30EmitsImportableSynonymResources(t *testing.T) {
	g, cleanup := newGeneratorForTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/synonym_sets" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"products","items":[{"id":"shoe terms","synonyms":["shoe","sneaker"]}]}]`))
	})
	defer cleanup()

	g.serverVersion = version.MustParse("30.0")
	g.featureChecker = version.NewFeatureChecker(g.serverVersion)

	f := hclwrite.NewEmptyFile()
	resourceNames := make(map[string]bool)
	collectionResourceMap := make(map[string]string)
	var importCommands []ImportCommand

	if err := g.generateSynonyms(context.Background(), f, resourceNames, collectionResourceMap, &importCommands); err != nil {
		t.Fatalf("generateSynonyms() returned error: %v", err)
	}

	hcl := string(f.Bytes())
	if !strings.Contains(hcl, `resource "`+tfnames.FullTypeName(tfnames.ResourceSynonym)+`"`) {
		t.Fatalf("generated HCL did not contain synonym resource:\n%s", hcl)
	}
	if !strings.Contains(hcl, `collection = "products"`) {
		t.Fatalf("generated HCL did not contain literal synonym set name:\n%s", hcl)
	}
	if len(importCommands) != 1 {
		t.Fatalf("generateSynonyms() produced %d import commands, want 1", len(importCommands))
	}
	if importCommands[0].ImportID != "products/shoe terms" {
		t.Fatalf("synonym import ID = %q, want %q", importCommands[0].ImportID, "products/shoe terms")
	}
}

func TestGenerateCurationSetsV30EmitsImportableOverrideResources(t *testing.T) {
	g, cleanup := newGeneratorForTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/curation_sets" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"products","items":[{"id":"featured","rule":{"query":"sale","match":"exact"}}]}]`))
	})
	defer cleanup()

	g.serverVersion = version.MustParse("30.0")
	g.featureChecker = version.NewFeatureChecker(g.serverVersion)

	f := hclwrite.NewEmptyFile()
	resourceNames := make(map[string]bool)
	collectionResourceMap := make(map[string]string)
	var importCommands []ImportCommand

	if err := g.generateOverrides(context.Background(), f, resourceNames, collectionResourceMap, &importCommands); err != nil {
		t.Fatalf("generateOverrides() returned error: %v", err)
	}

	hcl := string(f.Bytes())
	if !strings.Contains(hcl, `resource "`+tfnames.FullTypeName(tfnames.ResourceOverride)+`"`) {
		t.Fatalf("generated HCL did not contain override resource:\n%s", hcl)
	}
	if !strings.Contains(hcl, `collection = "products"`) {
		t.Fatalf("generated HCL did not contain literal curation set name:\n%s", hcl)
	}
	if len(importCommands) != 1 {
		t.Fatalf("generateOverrides() produced %d import commands, want 1", len(importCommands))
	}
	if importCommands[0].ImportID != "products/featured" {
		t.Fatalf("override import ID = %q, want %q", importCommands[0].ImportID, "products/featured")
	}
}

func TestDocumentExportURLEscapesCollectionName(t *testing.T) {
	got := documentExportURL("http", "127.0.0.1", 8108, "docs / prod")
	want := "http://127.0.0.1:8108/collections/docs%20%2F%20prod/documents/export"
	if got != want {
		t.Fatalf("documentExportURL() = %q, want %q", got, want)
	}
}

func TestClusterMatchesHostNormalizesHostnames(t *testing.T) {
	cluster := client.Cluster{
		Hostnames: client.ClusterHostnames{
			SearchDeliveryNetwork: "abc123.a1.typesense.net",
		},
	}

	if !clusterMatchesHost(&cluster, "HTTPS://ABC123.A1.TYPESENSE.NET:443") {
		t.Fatal("expected clusterMatchesHost to normalize scheme, case, and port")
	}
}

func TestClusterHostnameSummary(t *testing.T) {
	cluster := client.Cluster{
		ID:   "clu_123",
		Name: "docs-prod",
		Hostnames: client.ClusterHostnames{
			SearchDeliveryNetwork: "docs.a1.typesense.net",
			LoadBalanced:          "docs.a1.typesense.net",
			Nodes:                 []string{"docs-1.a1.typesense.net", "docs-2.a1.typesense.net"},
		},
	}

	got := clusterHostnameSummary(&cluster)
	want := `docs-prod (clu_123): search_delivery_network="docs.a1.typesense.net", load_balanced="docs.a1.typesense.net", nodes=["docs-1.a1.typesense.net", "docs-2.a1.typesense.net"]`
	if got != want {
		t.Fatalf("clusterHostnameSummary() = %q, want %q", got, want)
	}
}

func TestClusterHostnameInventory(t *testing.T) {
	clusters := []client.Cluster{
		{
			ID:   "clu_123",
			Name: "docs-prod",
			Hostnames: client.ClusterHostnames{
				SearchDeliveryNetwork: "docs.a1.typesense.net",
				LoadBalanced:          "docs.a1.typesense.net",
				Nodes:                 []string{"docs-1.a1.typesense.net"},
			},
		},
		{
			ID:   "clu_456",
			Name: "docs-staging",
		},
	}

	got := clusterHostnameInventory(clusters)
	want := `docs-prod (clu_123): search_delivery_network="docs.a1.typesense.net", load_balanced="docs.a1.typesense.net", nodes=["docs-1.a1.typesense.net"]; docs-staging (clu_456): search_delivery_network=<empty>, load_balanced=<empty>, nodes=[]`
	if got != want {
		t.Fatalf("clusterHostnameInventory() = %q, want %q", got, want)
	}
}

func TestClusterHostnameInventoryEmpty(t *testing.T) {
	got := clusterHostnameInventory(nil)
	if got != "<no clusters returned>" {
		t.Fatalf("clusterHostnameInventory(nil) = %q, want %q", got, "<no clusters returned>")
	}
}

func TestCollectionFingerprintSortsCollectionNames(t *testing.T) {
	collections := []client.Collection{
		{Name: "b"},
		{Name: "a"},
		{Name: "c"},
	}

	got := collectionFingerprint(collections)
	want := "a\x00b\x00c"
	if got != want {
		t.Fatalf("collectionFingerprint() = %q, want %q", got, want)
	}
}

func TestFindClustersByServerProbe(t *testing.T) {
	clusters := []client.Cluster{
		{
			ID:   "clu_123",
			Name: "docs",
			Hostnames: client.ClusterHostnames{
				SearchDeliveryNetwork: "docs.a1.typesense.net",
				Nodes:                 []string{"docs-1.a1.typesense.net"},
			},
		},
		{
			ID:   "clu_456",
			Name: "other",
			Hostnames: client.ClusterHostnames{
				SearchDeliveryNetwork: "other.a1.typesense.net",
			},
		},
	}

	expected := collectionFingerprint([]client.Collection{{Name: "typesense_docs"}, {Name: "typesense_docs_queries"}})
	probeMatches, matchedHosts := findClustersByServerProbe(context.Background(), clusters, expected, func(_ context.Context, host string) ([]client.Collection, error) {
		switch host {
		case "docs.a1.typesense.net":
			return []client.Collection{{Name: "typesense_docs_queries"}, {Name: "typesense_docs"}}, nil
		case "other.a1.typesense.net":
			return nil, errors.New("unauthorized")
		default:
			return nil, errors.New("unexpected host")
		}
	})

	if len(probeMatches) != 1 {
		t.Fatalf("findClustersByServerProbe() matched %d clusters, want 1", len(probeMatches))
	}
	if probeMatches[0].ID != "clu_123" {
		t.Fatalf("findClustersByServerProbe() matched cluster %q, want %q", probeMatches[0].ID, "clu_123")
	}
	if matchedHosts["clu_123"] != "docs.a1.typesense.net" {
		t.Fatalf("findClustersByServerProbe() matched host %q, want %q", matchedHosts["clu_123"], "docs.a1.typesense.net")
	}
}

func TestFileSetSingleFile(t *testing.T) {
	fs := newFileSet(true)

	mainFile := fs.get("main.tf")
	clusterFile := fs.get("cluster.tf")
	collectionsFile := fs.get("collections.tf")

	if mainFile != clusterFile {
		t.Error("in single-file mode, cluster.tf should return the same file as main.tf")
	}
	if mainFile != collectionsFile {
		t.Error("in single-file mode, collections.tf should return the same file as main.tf")
	}
	if len(fs.files) != 1 {
		t.Errorf("in single-file mode, expected 1 file in map, got %d", len(fs.files))
	}
}

func TestFileSetMultiFile(t *testing.T) {
	fs := newFileSet(false)

	mainFile := fs.get("main.tf")
	clusterFile := fs.get("cluster.tf")
	collectionsFile := fs.get("collections.tf")

	if mainFile == clusterFile {
		t.Error("in multi-file mode, cluster.tf should be a different file from main.tf")
	}
	if mainFile == collectionsFile {
		t.Error("in multi-file mode, collections.tf should be a different file from main.tf")
	}
	if clusterFile == collectionsFile {
		t.Error("in multi-file mode, cluster.tf should be a different file from collections.tf")
	}
	if len(fs.files) != 3 {
		t.Errorf("in multi-file mode, expected 3 files in map, got %d", len(fs.files))
	}
}

func TestFileSetGetIdempotent(t *testing.T) {
	fs := newFileSet(false)

	first := fs.get("cluster.tf")
	second := fs.get("cluster.tf")

	if first != second {
		t.Error("get() should return the same file for the same name")
	}
}
