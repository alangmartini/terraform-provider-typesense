//go:build e2e

package chinooktest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGenerateIdempotent applies chinook, runs `generate` twice into
// separate directories against the same populated cluster, and asserts
// the two outputs are byte-identical after normalizing the header
// timestamp. Non-determinism beyond the timestamp is a generator bug.
func TestGenerateIdempotent(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tf := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply chinook: %v", err)
	}

	dirA := t.TempDir()
	if err := runGenerate(t, cluster, dirA); err != nil {
		t.Fatalf("first generate: %v", err)
	}

	dirB := t.TempDir()
	if err := runGenerate(t, cluster, dirB); err != nil {
		t.Fatalf("second generate: %v", err)
	}

	filesA := listFiles(t, dirA)
	filesB := listFiles(t, dirB)

	if diff := stringSliceDiff(filesA, filesB); diff != "" {
		t.Fatalf("file sets differ between generate runs:\n%s", diff)
	}

	for _, name := range filesA {
		a := normalizeGeneratedFile(readFile(t, filepath.Join(dirA, name)))
		b := normalizeGeneratedFile(readFile(t, filepath.Join(dirB, name)))
		if a != b {
			t.Errorf("generate output for %s is not idempotent\n--- run 1 ---\n%s\n--- run 2 ---\n%s",
				name, a, b)
		}
	}
}

// generatedTimestampPattern matches the "Generated at" header line emitted
// by the generator, which embeds time.Now() and is the only documented
// source of non-determinism.
var generatedTimestampPattern = regexp.MustCompile(`(?m)^# Generated at: .+$`)

func normalizeGeneratedFile(content string) string {
	return generatedTimestampPattern.ReplaceAllString(content, "# Generated at: <NORMALIZED>")
}

// listFiles returns the sorted relative paths of all regular files under
// dir. Directories themselves are not included; nested files are joined
// with forward slashes for stable comparison.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("listFiles %s: %v", dir, err)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func stringSliceDiff(a, b []string) string {
	if equalStringSlices(a, b) {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("only in run 1: ")
	sb.WriteString(strings.Join(setDiff(a, b), ", "))
	sb.WriteString("\nonly in run 2: ")
	sb.WriteString(strings.Join(setDiff(b, a), ", "))
	return sb.String()
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func setDiff(a, b []string) []string {
	bset := make(map[string]struct{}, len(b))
	for _, s := range b {
		bset[s] = struct{}{}
	}
	var diff []string
	for _, s := range a {
		if _, ok := bset[s]; !ok {
			diff = append(diff, s)
		}
	}
	return diff
}
