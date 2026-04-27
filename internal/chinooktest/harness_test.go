//go:build e2e

// Package chinooktest contains end-to-end tests for the Typesense Terraform
// provider, exercised through the chinook example. Build tag `e2e` keeps these
// tests out of the default `go test ./...` run.
package chinooktest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

const (
	containerNamePrefix = "chinooke2e"
	defaultAPIKey       = "chinook-e2e-key"
	healthTimeout       = 30 * time.Second
)

// Cluster represents a running Typesense container under test.
type Cluster struct {
	Host    string
	Port    int
	APIKey  string
	BaseURL string
	Name    string
}

// Client returns a configured ServerClient for the cluster.
func (c *Cluster) Client() *client.ServerClient {
	return client.NewServerClient(c.Host, c.APIKey, c.Port, "http")
}

// StartCluster launches a fresh Typesense container at the given image tag,
// waits for /health to return 200, and registers cleanup that removes both
// the container and its data volume when the test finishes.
func StartCluster(t *testing.T, version string) *Cluster {
	t.Helper()

	port, err := freePort()
	if err != nil {
		t.Fatalf("StartCluster: %v", err)
	}

	suffix, err := randomSuffix()
	if err != nil {
		t.Fatalf("StartCluster: %v", err)
	}
	name := containerNamePrefix + "-" + suffix

	cluster := &Cluster{
		Host:    "127.0.0.1",
		Port:    port,
		APIKey:  defaultAPIKey,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		Name:    name,
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--add-host=host.docker.internal:host-gateway",
		"-p", fmt.Sprintf("%d:8108", port),
		"-v", name + "-data:/data",
		"typesense/typesense:" + version,
		"--data-dir=/data",
		"--api-key=" + defaultAPIKey,
		"--enable-cors",
	}
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), "MSYS_NO_PATHCONV=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("StartCluster: docker run failed: %v\n%s", err, out)
	}

	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
		_ = exec.Command("docker", "volume", "rm", name+"-data").Run()
	})

	if err := waitForHealth(cluster, healthTimeout); err != nil {
		logs, _ := exec.Command("docker", "logs", name).CombinedOutput()
		t.Fatalf("StartCluster: cluster %s did not become healthy: %v\nlogs:\n%s", name, err, logs)
	}

	return cluster
}

func waitForHealth(c *Cluster, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for /health from %s", c.BaseURL)
		default:
		}
		resp, err := http.Get(c.BaseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func randomSuffix() (string, error) {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("rand: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
