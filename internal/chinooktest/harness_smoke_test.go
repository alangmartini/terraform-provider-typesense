//go:build e2e

package chinooktest

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

// TestHarnessSmoke exercises StartCluster end-to-end: launches a Typesense
// container, asserts /health and the typed client both work, then verifies
// that the test cleanup actually removed the container and its volume.
func TestHarnessSmoke(t *testing.T) {
	var (
		name   string
		volume string
	)

	t.Run("starts healthy and exposes client", func(t *testing.T) {
		cluster := StartCluster(t, "30.1")
		name = cluster.Name
		volume = cluster.Name + "-data"

		resp, err := http.Get(cluster.BaseURL + "/health")
		if err != nil {
			t.Fatalf("GET %s/health: %v", cluster.BaseURL, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("/health status = %d, want 200", resp.StatusCode)
		}

		colls, err := cluster.Client().ListCollections(context.Background())
		if err != nil {
			t.Fatalf("ListCollections: %v", err)
		}
		if len(colls) != 0 {
			t.Errorf("fresh cluster has %d collections, want 0", len(colls))
		}
	})

	if leaked := dockerHasContainer(t, name); leaked {
		t.Errorf("container %s still exists after cleanup", name)
	}
	if leaked := dockerHasVolume(t, volume); leaked {
		t.Errorf("volume %s still exists after cleanup", volume)
	}
}

func dockerHasContainer(t *testing.T, name string) bool {
	t.Helper()
	out, err := exec.Command("docker", "ps", "-aq", "--filter", "name="+name).CombinedOutput()
	if err != nil {
		t.Fatalf("docker ps -aq: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out)) != ""
}

func dockerHasVolume(t *testing.T, name string) bool {
	t.Helper()
	out, err := exec.Command("docker", "volume", "ls", "-q", "--filter", "name="+name).CombinedOutput()
	if err != nil {
		t.Fatalf("docker volume ls: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}
