package generator

import (
	"context"
	"errors"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
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
