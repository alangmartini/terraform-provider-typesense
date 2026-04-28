//go:build e2e

package chinooktest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestImportRoundtrip applies chinook to a fresh cluster, runs `generate`
// against the populated cluster, then re-applies the generated config in a
// fresh state and asserts `terraform plan` reports zero changes.
//
// nl_search_models.tf is dropped from the generated output before re-apply
// because Typesense never returns the model's api_key, so a generated
// resource that references it via var.openai_api_key cannot round-trip
// without a non-zero diff.
func TestImportRoundtrip(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)
	m := MaterializeChinook(t, "30.1", MaterializeOptions{MockOpenAIURL: mock.URL})

	tfA := NewTerraform(t, m.Dir)
	vars := chinookVars(cluster, m.Vars)

	if err := tfA.Apply(vars); err != nil {
		t.Fatalf("apply chinook (state A): %v", err)
	}

	genDir := t.TempDir()
	if err := runGenerate(t, cluster, genDir); err != nil {
		t.Fatalf("generate: %v", err)
	}

	stripUnrestorableResources(t, genDir)

	// The generated main.tf comments out server_api_key; the provider falls
	// back to TYPESENSE_API_KEY when the attribute is unset.
	t.Setenv("TYPESENSE_API_KEY", cluster.APIKey)

	tfB := NewTerraform(t, genDir)
	if err := tfB.Apply(nil); err != nil {
		t.Fatalf("apply generated config (state B): %v", err)
	}

	code, planOut, err := tfB.PlanWithOutput(nil)
	if err != nil {
		t.Fatalf("plan generated config: %v\noutput:\n%s", err, planOut)
	}
	if code != 0 {
		t.Errorf("plan exit code = %d, want 0 (no changes)\noutput:\n%s", code, planOut)
	}
}

func runGenerate(t *testing.T, c *Cluster, outputDir string) error {
	t.Helper()
	bin := filepath.Join(providerBinaryDir, "terraform-provider-typesense")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command(bin, "generate",
		"--host", c.Host,
		"--port", fmt.Sprintf("%d", c.Port),
		"--protocol", "http",
		"--api-key", c.APIKey,
		"--output", outputDir,
		"--include-data",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("generate: %w\n%s", err, out)
	}
	return nil
}

// stripUnrestorableResources removes generated files for resources whose
// state cannot round-trip without divergence (e.g., nl_search_models, whose
// api_key is not returned by the API).
func stripUnrestorableResources(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{"nl_search_models.tf"} {
		_ = os.Remove(filepath.Join(dir, name))
	}

	importsPath := filepath.Join(dir, "imports.tf")
	body, err := os.ReadFile(importsPath)
	if err != nil {
		t.Fatalf("read imports.tf: %v", err)
	}
	filtered := dropImportsForResourceType(string(body), "typesense_nl_search_model")
	if err := os.WriteFile(importsPath, []byte(filtered), 0o600); err != nil {
		t.Fatalf("write imports.tf: %v", err)
	}
}

// dropImportsForResourceType removes import { ... } blocks whose `to = X.Y`
// references a given resource type.
func dropImportsForResourceType(content, resourceType string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))

	i := 0
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "import {" {
			blockEnd := i
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "}" {
					blockEnd = j
					break
				}
			}

			matches := false
			for j := i; j <= blockEnd; j++ {
				if strings.Contains(lines[j], "to = "+resourceType+".") {
					matches = true
					break
				}
			}

			if matches {
				i = blockEnd + 1
				if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
					i++
				}
				continue
			}
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n")
}

