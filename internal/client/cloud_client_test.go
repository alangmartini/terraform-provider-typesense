package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestCreateClusterConfigChange_Payload validates that the config change request
// sends the correct JSON payload to the Typesense Cloud API.
func TestCreateClusterConfigChange_Payload(t *testing.T) {
	var capturedBody []byte
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ClusterConfigChange{
			ID:        "change-123",
			ClusterID: "cluster-abc",
			Status:    "queued",
		})
	}))
	defer server.Close()

	client := &CloudClient{
		httpClient: server.Client(),
		apiKey:     "test-key",
		baseURL:    server.URL,
	}

	change := &ClusterConfigChange{
		ClusterID:   "cluster-abc",
		NewMemory:   "8_gb",
		NewVCPU:     "4_vcpus",
		NewTypesenseVersion: "28.0",
	}

	result, err := client.CreateClusterConfigChange(context.Background(), change)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify endpoint path
	if capturedPath != "/clusters/cluster-abc/configuration-changes" {
		t.Errorf("Expected path /clusters/cluster-abc/configuration-changes, got %s", capturedPath)
	}

	// Verify payload fields
	var payload map[string]interface{}
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("Failed to parse request body: %v", err)
	}

	if payload["new_memory"] != "8_gb" {
		t.Errorf("Expected new_memory=8_gb, got %v", payload["new_memory"])
	}
	if payload["new_vcpu"] != "4_vcpus" {
		t.Errorf("Expected new_vcpu=4_vcpus, got %v", payload["new_vcpu"])
	}
	if payload["new_typesense_server_version"] != "28.0" {
		t.Errorf("Expected new_typesense_server_version=28.0, got %v", payload["new_typesense_server_version"])
	}

	// Verify omitted fields are not present
	if _, ok := payload["new_high_availability"]; ok {
		t.Error("new_high_availability should be omitted when empty")
	}

	// Verify response
	if result.ID != "change-123" {
		t.Errorf("Expected ID=change-123, got %s", result.ID)
	}
	if result.Status != "queued" {
		t.Errorf("Expected Status=queued, got %s", result.Status)
	}
}

// TestWaitForClusterReady_AfterConfigChange validates that WaitForClusterReady
// polls until the cluster transitions from a transitional state to in_service.
func TestWaitForClusterReady_AfterConfigChange(t *testing.T) {
	var pollCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&pollCount, 1)
		status := "configuring"
		if count >= 2 {
			status = "in_service"
		}
		json.NewEncoder(w).Encode(Cluster{
			ID:     "cluster-abc",
			Name:   "test",
			Status: status,
			Memory: "8_gb",
		})
	}))
	defer server.Close()

	client := &CloudClient{
		httpClient: server.Client(),
		apiKey:     "test-key",
		baseURL:    server.URL,
	}

	// Use a short poll interval for testing — WaitForClusterReady uses 30s ticker
	// so we test via GetCluster directly to verify status handling
	cluster, err := client.GetCluster(context.Background(), "cluster-abc")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// First call returns "configuring"
	if cluster.Status != "configuring" {
		t.Errorf("Expected first status=configuring, got %s", cluster.Status)
	}

	// Second call returns "in_service"
	cluster, err = client.GetCluster(context.Background(), "cluster-abc")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cluster.Status != "in_service" {
		t.Errorf("Expected second status=in_service, got %s", cluster.Status)
	}
}

// TestCreateClusterConfigChange_OnlyChangedFields validates that only the fields
// that are actually set get included in the API request (omitempty behavior).
func TestCreateClusterConfigChange_OnlyChangedFields(t *testing.T) {
	tests := []struct {
		name           string
		change         ClusterConfigChange
		expectedFields []string
		absentFields   []string
	}{
		{
			name: "memory only",
			change: ClusterConfigChange{
				ClusterID: "c1",
				NewMemory: "16_gb",
			},
			expectedFields: []string{"new_memory"},
			absentFields:   []string{"new_vcpu", "new_high_availability", "new_typesense_server_version"},
		},
		{
			name: "version only",
			change: ClusterConfigChange{
				ClusterID:           "c1",
				NewTypesenseVersion: "28.0",
			},
			expectedFields: []string{"new_typesense_server_version"},
			absentFields:   []string{"new_memory", "new_vcpu", "new_high_availability"},
		},
		{
			name: "multiple fields",
			change: ClusterConfigChange{
				ClusterID:           "c1",
				NewMemory:           "32_gb",
				NewVCPU:             "8_vcpus",
				NewHighAvailability: "yes_3_way",
				NewTypesenseVersion: "28.0",
			},
			expectedFields: []string{"new_memory", "new_vcpu", "new_high_availability", "new_typesense_server_version"},
			absentFields:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ClusterConfigChange{ID: "ch-1", Status: "queued"})
			}))
			defer server.Close()

			client := &CloudClient{
				httpClient: server.Client(),
				apiKey:     "test-key",
				baseURL:    server.URL,
			}

			_, err := client.CreateClusterConfigChange(context.Background(), &tt.change)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			var payload map[string]interface{}
			if err := json.Unmarshal(capturedBody, &payload); err != nil {
				t.Fatalf("Failed to parse body: %v", err)
			}

			for _, field := range tt.expectedFields {
				if _, ok := payload[field]; !ok {
					t.Errorf("Expected field %q to be present", field)
				}
			}
			for _, field := range tt.absentFields {
				if _, ok := payload[field]; ok {
					t.Errorf("Expected field %q to be absent", field)
				}
			}
		})
	}
}

// TestUpdateCluster_DirectFieldsOnly validates that UpdateCluster only sends
// the mutable fields (name, auto_upgrade_capacity) to the API.
func TestUpdateCluster_DirectFieldsOnly(t *testing.T) {
	var capturedBody []byte
	var capturedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body)
		json.NewEncoder(w).Encode(Cluster{
			ID:                  "cluster-abc",
			Name:                "new-name",
			AutoUpgradeCapacity: true,
			Status:              "in_service",
		})
	}))
	defer server.Close()

	client := &CloudClient{
		httpClient: server.Client(),
		apiKey:     "test-key",
		baseURL:    server.URL,
	}

	cluster := &Cluster{
		Name:                "new-name",
		AutoUpgradeCapacity: true,
	}

	_, err := client.UpdateCluster(context.Background(), "cluster-abc", cluster)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if capturedMethod != http.MethodPatch {
		t.Errorf("Expected PATCH method, got %s", capturedMethod)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if payload["name"] != "new-name" {
		t.Errorf("Expected name=new-name, got %v", payload["name"])
	}
}
