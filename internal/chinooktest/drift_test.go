//go:build e2e

package chinooktest

import (
	"context"
	"testing"
)

// TestDrift applies chinook, mutates state directly via the client (deletes
// a stopwords set), confirms `terraform plan` reports drift, runs apply to
// restore, and verifies the resource is back.
func TestDrift(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("first apply: %v", err)
	}

	cli := cluster.Client()
	ctx := context.Background()

	if set, err := cli.GetStopwordsSet(ctx, "english-common"); err != nil || set == nil {
		t.Fatalf("english-common missing before drift: set=%v err=%v", set, err)
	}

	if err := cli.DeleteStopwordsSet(ctx, "english-common"); err != nil {
		t.Fatalf("DeleteStopwordsSet (drift inject): %v", err)
	}

	code, err := tf.Plan(vars)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if code != 2 {
		t.Errorf("plan exit code = %d, want 2 (changes pending)", code)
	}

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply (drift recovery): %v", err)
	}

	set, err := cli.GetStopwordsSet(ctx, "english-common")
	if err != nil {
		t.Fatalf("GetStopwordsSet after recovery: %v", err)
	}
	if set == nil {
		t.Errorf("english-common still missing after recovery apply")
	}

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}
}
