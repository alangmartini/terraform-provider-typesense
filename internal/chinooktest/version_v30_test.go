//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestVersionV30 exercises the full chinook fixture on the latest pinned
// v30 image, covering synonym sets, curation sets, NL search models,
// stemming dictionaries, and analytics rules. Symmetry with the
// per-version tier — TestApply already exercises this version.
func TestVersionV30(t *testing.T) {
	runChinookVersion(t, versionScenario{
		Image: "30.1",
		Verify: func(t *testing.T, cli *client.ServerClient) {
			ctx := context.Background()

			expectCount(t, "collections", 10, func() (int, error) {
				c, err := cli.ListCollections(ctx)
				return len(c), err
			})
			expectCount(t, "aliases", 6, func() (int, error) {
				a, err := cli.ListCollectionAliases(ctx)
				return len(a), err
			})
			expectCount(t, "presets", 12, func() (int, error) {
				p, err := cli.ListPresets(ctx)
				return len(p), err
			})
			expectCount(t, "stopwords sets", 3, func() (int, error) {
				s, err := cli.ListStopwordsSets(ctx)
				return len(s), err
			})
			expectCount(t, "analytics rules", 3, func() (int, error) {
				a, err := cli.ListAnalyticsRules(ctx)
				return len(a), err
			})
			expectCount(t, "stemming dictionaries", 1, func() (int, error) {
				d, err := cli.ListStemmingDictionaries(ctx)
				return len(d), err
			})
			expectCount(t, "nl search models", 1, func() (int, error) {
				n, err := cli.ListNLSearchModels(ctx)
				return len(n), err
			})

			synSets, err := cli.ListSynonymSets(ctx)
			if err != nil {
				t.Errorf("list synonym sets: %v", err)
			} else {
				total := 0
				for _, s := range synSets {
					total += len(s.Synonyms)
				}
				if total != 20 {
					t.Errorf("synonym set items: got %d, want 20", total)
				}
			}

			curSets, err := cli.ListCurationSets(ctx)
			if err != nil {
				t.Errorf("list curation sets: %v", err)
			} else {
				total := 0
				for _, c := range curSets {
					total += len(c.Curations)
				}
				if total != 9 {
					t.Errorf("curation set items: got %d, want 9", total)
				}
			}
		},
	})
}
