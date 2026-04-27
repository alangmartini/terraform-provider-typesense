//go:build e2e

package chinooktest

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestUpdate applies chinook, mutates a stopwords list in the materialized
// .tf, re-applies, and asserts the mutation reaches the cluster.
func TestUpdate(t *testing.T) {
	const sentinel = "e2eupdate"

	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	addStopwordToEnglishCommon(t, m.Dir, sentinel)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	cli := cluster.Client()
	set, err := cli.GetStopwordsSet(context.Background(), "english-common")
	if err != nil {
		t.Fatalf("GetStopwordsSet: %v", err)
	}
	if set == nil {
		t.Fatalf("english-common stopwords set not found")
	}
	if !slices.Contains(set.Stopwords, sentinel) {
		t.Errorf("sentinel %q not in stopwords after update: %v", sentinel, set.Stopwords)
	}

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}
}

// addStopwordToEnglishCommon edits stopwords.tf in place, appending the given
// sentinel word to the english_common stopwords list.
func addStopwordToEnglishCommon(t *testing.T, dir, sentinel string) {
	t.Helper()
	path := filepath.Join(dir, "stopwords.tf")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stopwords.tf: %v", err)
	}

	anchor := []byte(`"been",`)
	if !bytes.Contains(body, anchor) {
		t.Fatalf("anchor %q not found in stopwords.tf", anchor)
	}
	replacement := append([]byte(nil), anchor...)
	replacement = append(replacement, []byte("\n    \""+sentinel+"\",")...)
	body = bytes.Replace(body, anchor, replacement, 1)

	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write stopwords.tf: %v", err)
	}
}
