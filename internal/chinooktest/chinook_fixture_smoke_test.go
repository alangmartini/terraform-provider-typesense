//go:build e2e

package chinooktest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMaterializeChinookV27DropsModernResources verifies that chinook
// resource files unsupported on v27 are not copied into the materialized
// directory.
func TestMaterializeChinookV27DropsModernResources(t *testing.T) {
	m := MaterializeChinook(t, "27.1", MaterializeOptions{})

	mustNotExist(t, m.Dir, "analytics.tf")
	mustNotExist(t, m.Dir, "stemming.tf")
	mustNotExist(t, m.Dir, "nl_search_model.tf")
	mustNotExist(t, m.Dir, "conversation_model.tf")

	mustExist(t, m.Dir, "main.tf")
	mustExist(t, m.Dir, "collections.tf")
	mustExist(t, m.Dir, "stopwords.tf")
	mustExist(t, m.Dir, "presets.tf")
	mustExist(t, m.Dir, "synonyms.tf")
	mustExist(t, m.Dir, "curations.tf")
	mustExist(t, m.Dir, "aliases.tf")
}

// TestMaterializeChinookV30KeepsAllResources verifies the v30 materialization
// keeps every resource file from the chinook example. conversation_model.tf
// is intentionally excluded: see chinookSkip for the reason.
func TestMaterializeChinookV30KeepsAllResources(t *testing.T) {
	m := MaterializeChinook(t, "30.1", MaterializeOptions{})

	for _, name := range []string{
		"main.tf",
		"variables.tf",
		"collections.tf",
		"aliases.tf",
		"stopwords.tf",
		"presets.tf",
		"synonyms.tf",
		"curations.tf",
		"analytics.tf",
		"stemming.tf",
		"nl_search_model.tf",
		"api_keys.tf",
	} {
		mustExist(t, m.Dir, name)
	}

	mustNotExist(t, m.Dir, "conversation_model.tf")
}

// TestMaterializeChinookSkipsState verifies stray terraform state files are
// not copied into the materialized directory.
func TestMaterializeChinookSkipsState(t *testing.T) {
	m := MaterializeChinook(t, "30.1", MaterializeOptions{})

	mustNotExist(t, m.Dir, "terraform.tfstate")
	mustNotExist(t, m.Dir, "terraform.tfstate.backup")
	mustNotExist(t, m.Dir, ".terraform")
	mustNotExist(t, m.Dir, ".terraform.lock.hcl")
}

// TestMaterializeChinookProvidesMockVars verifies opts.MockOpenAIURL flows
// into the Vars map under the mock_openai_url key.
func TestMaterializeChinookProvidesMockVars(t *testing.T) {
	mockURL := "http://host.docker.internal:9999"
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mockURL})

	if got := m.Vars["mock_openai_url"]; got != mockURL {
		t.Errorf("Vars[mock_openai_url] = %q, want %q", got, mockURL)
	}
}

func mustExist(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Errorf("expected %s to exist in materialized chinook: %v", name, err)
	}
}

func mustNotExist(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		t.Errorf("expected %s NOT to exist in materialized chinook", name)
	} else if !os.IsNotExist(err) {
		t.Errorf("stat %s: %v", name, err)
	}
}
