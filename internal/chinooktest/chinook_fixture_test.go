//go:build e2e

package chinooktest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/version"
)

// MaterializeOptions controls per-version filtering of the chinook example.
type MaterializeOptions struct {
	// MockOpenAIURL, when non-empty, is forwarded as -var=mock_openai_url
	// during apply/destroy so NL/conversation model resources hit a local
	// mock instead of api.openai.com.
	MockOpenAIURL string
}

// Materialized describes a chinook example checkout tailored to a target
// Typesense version. Tests pass Vars to the Terraform runner.
type Materialized struct {
	Dir  string
	Vars map[string]string
}

// MaterializeChinook copies examples/chinook into a per-test temp dir and
// drops any resource files unsupported by the target Typesense version.
// Stray state files (.terraform, terraform.tfstate, lock files) are skipped.
func MaterializeChinook(t *testing.T, ver string, opts MaterializeOptions) *Materialized {
	t.Helper()

	root, err := repoRoot()
	if err != nil {
		t.Fatalf("MaterializeChinook: repo root: %v", err)
	}
	src := filepath.Join(root, "examples", "chinook")

	dst := t.TempDir()
	if err := copyChinookDir(src, dst, chinookSkip(ver)); err != nil {
		t.Fatalf("MaterializeChinook: copy: %v", err)
	}

	vars := map[string]string{}
	if opts.MockOpenAIURL != "" {
		vars["mock_openai_url"] = opts.MockOpenAIURL
	}

	return &Materialized{Dir: dst, Vars: vars}
}

func chinookSkip(ver string) map[string]bool {
	v, err := version.Parse(ver)
	if err != nil {
		return nil
	}
	checker := version.NewFeatureChecker(v)

	skip := make(map[string]bool)
	if !checker.SupportsFeature(version.FeatureAnalyticsRules) {
		skip["analytics.tf"] = true
	}
	if !checker.SupportsFeature(version.FeatureStemmingDictionaries) {
		skip["stemming.tf"] = true
	}
	if !checker.SupportsFeature(version.FeatureNLSearchModels) {
		skip["nl_search_model.tf"] = true
	}

	// outputs.tf references analytics_rules and nl_search_model resources
	// directly. Drop it whenever either is unavailable so the missing
	// references don't fail the plan. Outputs are diagnostic only and not
	// asserted by the e2e tests.
	if skip["analytics.tf"] || skip["nl_search_model.tf"] {
		skip["outputs.tf"] = true
	}

	// conversation_model.tf is dropped from materialized fixtures across
	// all versions: Typesense validates the model by calling the LLM, and
	// Phase 0 of the e2e plan only confirmed mock interception for
	// nl_search_models (api_url override). Restore once the mock covers
	// the conversation_model validation path.
	skip["conversation_model.tf"] = true

	return skip
}

func copyChinookDir(src, dst string, skip map[string]bool) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	for _, e := range entries {
		name := e.Name()
		if skipChinookEntry(name) || skip[name] {
			continue
		}
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)
		if e.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyChinookDir(srcPath, dstPath, nil); err != nil {
				return err
			}
			continue
		}
		if err := copyChinookFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func skipChinookEntry(name string) bool {
	switch name {
	case ".terraform", "terraform.tfstate", "terraform.tfstate.backup",
		".terraform.lock.hcl", "terraform.tfvars", "terraform.tfvars.json":
		return true
	}
	return false
}

func copyChinookFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
