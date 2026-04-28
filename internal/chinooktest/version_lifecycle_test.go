//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// chinookCollectionsWithSynonyms lists every chinook collection that has
// at least one synonym defined in synonyms.tf. Used to total per-collection
// synonyms on v27-v29 clusters.
var chinookCollectionsWithSynonyms = []string{"albums", "tracks", "playlists"}

// chinookCollectionsWithOverrides lists every chinook collection that has
// at least one override defined in curations.tf. Used for per-collection
// overrides on v27-v29 clusters.
var chinookCollectionsWithOverrides = []string{"tracks", "albums"}

// versionScenario configures a per-version chinook lifecycle test.
type versionScenario struct {
	// Image is the typesense Docker tag (e.g. "27.1", "28.0").
	Image string
	// Verify runs after a successful Apply, against the populated cluster.
	Verify func(t *testing.T, cli *client.ServerClient)
}

// runChinookVersion materializes chinook for the given version, applies,
// runs scenario.Verify, destroys, and asserts the cluster ends up empty.
func runChinookVersion(t *testing.T, scenario versionScenario) {
	t.Helper()

	cluster := StartCluster(t, scenario.Image)
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, scenario.Image, MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply: %v", err)
	}

	scenario.Verify(t, cluster.Client())

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}

	colls, err := cluster.Client().ListCollections(context.Background())
	if err != nil {
		t.Fatalf("list collections after destroy: %v", err)
	}
	if len(colls) != 0 {
		t.Errorf("after destroy: %d collections remain, want 0", len(colls))
	}
}

// expectPerCollectionSynonyms totals synonyms across the chinook collections
// known to define them, comparing the sum against want. Used on v27-v29.
func expectPerCollectionSynonyms(t *testing.T, cli *client.ServerClient, want int) {
	t.Helper()
	ctx := context.Background()
	total := 0
	for _, name := range chinookCollectionsWithSynonyms {
		s, err := cli.ListSynonyms(ctx, name)
		if err != nil {
			t.Errorf("list synonyms for %s: %v", name, err)
			return
		}
		total += len(s)
	}
	if total != want {
		t.Errorf("per-collection synonyms: got %d, want %d", total, want)
	}
}

// expectPerCollectionOverrides totals overrides across the chinook
// collections known to define them, comparing the sum against want.
func expectPerCollectionOverrides(t *testing.T, cli *client.ServerClient, want int) {
	t.Helper()
	ctx := context.Background()
	total := 0
	for _, name := range chinookCollectionsWithOverrides {
		o, err := cli.ListOverrides(ctx, name)
		if err != nil {
			t.Errorf("list overrides for %s: %v", name, err)
			return
		}
		total += len(o)
	}
	if total != want {
		t.Errorf("per-collection overrides: got %d, want %d", total, want)
	}
}
