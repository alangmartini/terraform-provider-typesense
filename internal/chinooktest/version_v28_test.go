//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestVersionV28 applies chinook against a v28 cluster. v28 introduces
// analytics rules and stemming dictionaries but predates NL search models
// (so nl_search_model.tf is dropped by the materializer). Synonyms and
// overrides are per-collection.
func TestVersionV28(t *testing.T) {
	runChinookVersion(t, versionScenario{
		Image: "28.0",
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

			expectPerCollectionSynonyms(t, cli, 20)
			expectPerCollectionOverrides(t, cli, 9)
		},
	})
}
