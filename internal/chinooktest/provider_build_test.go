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

// providerBinaryDir is the directory containing a freshly-built
// terraform-provider-typesense binary. Populated once by TestMain and
// referenced by NewTerraform when generating dev_overrides.
var providerBinaryDir string

// TestMain builds the provider binary once per `go test` invocation and
// exposes its directory via providerBinaryDir. The binary is built into a
// temp directory that is removed when the test process exits cleanly.
func TestMain(m *testing.M) {
	os.Exit(runTestMain(m))
}

func runTestMain(m *testing.M) int {
	dir, cleanup, err := buildProviderBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "chinooktest: %v\n", err)
		return 1
	}
	defer cleanup()
	providerBinaryDir = dir
	return m.Run()
}

func buildProviderBinary() (string, func(), error) {
	root, err := repoRoot()
	if err != nil {
		return "", nil, fmt.Errorf("locate repo root: %w", err)
	}

	// Build to a stable path under <repo>/bin so Windows Firewall remembers
	// the per-binary network access rule across test runs (avoiding repeated
	// "Allow this app" popups). The directory is gitignored via *.exe.
	dir := filepath.Join(root, "bin", "chinooktest")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("mkdir provider bindir: %w", err)
	}
	cleanup := func() {}

	bin := filepath.Join(dir, "terraform-provider-typesense")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("go build provider: %w\n%s", err, out)
	}
	return dir, cleanup, nil
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --show-toplevel: %w\n%s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}
