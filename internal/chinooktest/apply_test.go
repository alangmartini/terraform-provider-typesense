//go:build e2e

package chinooktest

import (
	"context"
	"fmt"
	"testing"
)

// TestApply applies the chinook example to a fresh Typesense v30 container,
// asserts every supported resource type exists with the expected cardinality,
// then destroys and confirms the cluster returns to an empty state.
func TestApply(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply: %v", err)
	}

	cli := cluster.Client()
	ctx := context.Background()

	expectCount(t, "collections", 10, func() (int, error) {
		c, err := cli.ListCollections(ctx)
		return len(c), err
	})
	expectCount(t, "aliases", 6, func() (int, error) {
		a, err := cli.ListCollectionAliases(ctx)
		return len(a), err
	})
	expectCount(t, "stopwords sets", 3, func() (int, error) {
		s, err := cli.ListStopwordsSets(ctx)
		return len(s), err
	})
	expectCount(t, "presets", 12, func() (int, error) {
		p, err := cli.ListPresets(ctx)
		return len(p), err
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

	keys, err := cli.ListAPIKeys(ctx)
	if err != nil {
		t.Errorf("list api keys: %v", err)
	} else if len(keys) < 3 {
		t.Errorf("api keys: got %d, want >= 3", len(keys))
	}

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}

	colls, err := cli.ListCollections(ctx)
	if err != nil {
		t.Fatalf("list collections after destroy: %v", err)
	}
	if len(colls) != 0 {
		t.Errorf("after destroy: %d collections remain, want 0", len(colls))
	}
}

func expectCount(t *testing.T, label string, want int, fetch func() (int, error)) {
	t.Helper()
	got, err := fetch()
	if err != nil {
		t.Errorf("list %s: %v", label, err)
		return
	}
	if got != want {
		t.Errorf("%s: got %d, want %d", label, got, want)
	}
}

// chinookVars composes the standard vars for applying chinook against a
// test cluster, merging in any extras from the materialized fixture.
func chinookVars(c *Cluster, extra map[string]string) map[string]string {
	vars := map[string]string{
		"typesense_host":     c.Host,
		"typesense_port":     fmt.Sprintf("%d", c.Port),
		"typesense_protocol": "http",
		"typesense_api_key":  c.APIKey,
		"openai_api_key":     "mock-openai-key",
	}
	for k, v := range extra {
		vars[k] = v
	}
	return vars
}
