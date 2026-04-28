//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestVersionV27 applies chinook against a v27 cluster. v27 lacks
// analytics rules, stemming dictionaries, and NL search models, so the
// materializer drops their files. Synonyms and overrides remain
// per-collection.
func TestVersionV27(t *testing.T) {
	runChinookVersion(t, versionScenario{
		Image: "27.1",
		Verify: func(t *testing.T, cli *client.ServerClient) {
			ctx := context.Background()

			// 7 instead of 10: analytics.tf is dropped, removing the
			// track_queries / album_queries / nohits_queries destination
			// collections.
			expectCount(t, "collections", 7, func() (int, error) {
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

			expectPerCollectionSynonyms(t, cli, 20)
			expectPerCollectionOverrides(t, cli, 9)
		},
	})
}
