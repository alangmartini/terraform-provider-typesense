//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestVersionV29 applies chinook against a v29 cluster. v29 supports NL
// search models but uses per-collection synonyms and overrides (synonym
// sets and curation sets are v30+). Stemming dictionaries and analytics
// rules are still supported.
func TestVersionV29(t *testing.T) {
	runChinookVersion(t, versionScenario{
		Image: "29.0",
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

			expectPerCollectionSynonyms(t, cli, 20)
			expectPerCollectionOverrides(t, cli, 9)
		},
	})
}
