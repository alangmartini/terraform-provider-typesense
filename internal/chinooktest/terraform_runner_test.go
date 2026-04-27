//go:build e2e

package chinooktest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Terraform wraps `terraform` CLI invocations against a working directory
// with isolated state and a generated dev-override config that points at the
// provider binary built by TestMain. Tests use one Terraform per workDir.
type Terraform struct {
	t       *testing.T
	binary  string
	workDir string
	state   string
	env     []string
}

// NewTerraform initializes a runner for workDir. It generates a per-test
// .terraformrc with dev_overrides for alanm/typesense, points
// TF_CLI_CONFIG_FILE at it, and resolves the terraform binary from
// TYPESENSE_E2E_TERRAFORM (if set) or PATH.
func NewTerraform(t *testing.T, workDir string) *Terraform {
	t.Helper()

	if providerBinaryDir == "" {
		t.Fatal("provider binary not built; TestMain did not run")
	}

	binary := os.Getenv("TYPESENSE_E2E_TERRAFORM")
	if binary == "" {
		binary = "terraform"
	}

	rcPath := filepath.Join(t.TempDir(), ".terraformrc")
	overrideDir := strings.ReplaceAll(providerBinaryDir, `\`, `/`)
	rc := fmt.Sprintf(`provider_installation {
  dev_overrides {
    "alanm/typesense" = "%s"
  }
  direct {}
}
`, overrideDir)
	if err := os.WriteFile(rcPath, []byte(rc), 0o600); err != nil {
		t.Fatalf("write .terraformrc: %v", err)
	}

	return &Terraform{
		t:       t,
		binary:  binary,
		workDir: workDir,
		state:   filepath.Join(workDir, "terraform.tfstate"),
		env:     append(os.Environ(), "TF_CLI_CONFIG_FILE="+rcPath),
	}
}

// Apply runs `terraform apply -auto-approve` against the working directory.
func (tf *Terraform) Apply(vars map[string]string) error {
	args := append([]string{"-auto-approve", "-state=" + tf.state}, varFlags(vars)...)
	return tf.run("apply", args)
}

// Destroy runs `terraform destroy -auto-approve`.
func (tf *Terraform) Destroy(vars map[string]string) error {
	args := append([]string{"-auto-approve", "-state=" + tf.state}, varFlags(vars)...)
	return tf.run("destroy", args)
}

// Plan runs `terraform plan -detailed-exitcode` and returns the exit code:
// 0 = no changes, 1 = error, 2 = changes pending. The error is non-nil on
// exit code 1; exit code 2 is reported as nil error with code=2.
func (tf *Terraform) Plan(vars map[string]string) (int, error) {
	args := append([]string{"-detailed-exitcode", "-state=" + tf.state}, varFlags(vars)...)
	return tf.runExit("plan", args)
}

func (tf *Terraform) run(subcmd string, args []string) error {
	_, err := tf.runExit(subcmd, args)
	return err
}

func (tf *Terraform) runExit(subcmd string, args []string) (int, error) {
	full := append([]string{subcmd}, args...)
	cmd := exec.Command(tf.binary, full...)
	cmd.Dir = tf.workDir
	cmd.Env = tf.env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if asExitErr(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			if subcmd == "plan" && exitCode == 2 {
				return exitCode, nil
			}
		}
		return exitCode, fmt.Errorf("terraform %s exit=%d: %w\nstdout:\n%s\nstderr:\n%s",
			subcmd, exitCode, err, stdout.String(), stderr.String())
	}
	return exitCode, nil
}

func asExitErr(err error, target **exec.ExitError) bool {
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	return false
}

func varFlags(vars map[string]string) []string {
	if len(vars) == 0 {
		return nil
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	flags := make([]string, 0, len(vars)*2)
	for _, k := range keys {
		flags = append(flags, "-var", fmt.Sprintf("%s=%s", k, vars[k]))
	}
	return flags
}
