// Package client provides HTTP clients for Typesense APIs
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	CloudAPIBaseURL = "https://cloud.typesense.org/api/v1"
)

// CloudClient handles communication with the Typesense Cloud Management API
type CloudClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewCloudClient creates a new Cloud Management API client
func NewCloudClient(apiKey string) *CloudClient {
	return &CloudClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:  apiKey,
		baseURL: CloudAPIBaseURL,
	}
}

// Cluster represents a Typesense Cloud cluster
type Cluster struct {
	ID                     string            `json:"id,omitempty"`
	Name                   string            `json:"name"`
	Memory                 string            `json:"memory"`
	VCPU                   string            `json:"vcpu"`
	HighAvailability       string            `json:"high_availability"`
	SearchDeliveryNetwork  string            `json:"search_delivery_network,omitempty"`
	TypesenseServerVersion string            `json:"typesense_server_version"`
	Regions                []string          `json:"regions"`
	Status                 string            `json:"status,omitempty"`
	Hostnames              ClusterHostnames  `json:"hostnames,omitempty"`
	APIKeys                *ClusterAPIKeys   `json:"api_keys,omitempty"`
	AutoUpgradeCapacity    bool              `json:"auto_upgrade_capacity,omitempty"`
	CreatedAt              string            `json:"created_at,omitempty"`
}

// ClusterHostnames contains cluster endpoint information
type ClusterHostnames struct {
	LoadBalanced string   `json:"load_balanced,omitempty"`
	Nodes        []string `json:"nodes,omitempty"`
}

// ClusterAPIKeys contains the cluster's API keys
type ClusterAPIKeys struct {
	Admin      string `json:"admin,omitempty"`
	Search     string `json:"search,omitempty"`
	SearchOnly string `json:"search_only,omitempty"`
}

// ClusterConfigChange represents a scheduled configuration change
type ClusterConfigChange struct {
	ID                     string `json:"id,omitempty"`
	ClusterID              string `json:"cluster_id"`
	NewMemory              string `json:"new_memory,omitempty"`
	NewVCPU                string `json:"new_vcpu,omitempty"`
	NewHighAvailability    string `json:"new_high_availability,omitempty"`
	NewTypesenseVersion    string `json:"new_typesense_server_version,omitempty"`
	PerformChangeAt        int64  `json:"perform_change_at,omitempty"`
	Status                 string `json:"status,omitempty"`
}

// CreateCluster creates a new Typesense Cloud cluster
func (c *CloudClient) CreateCluster(ctx context.Context, cluster *Cluster) (*Cluster, error) {
	body, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/clusters", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create cluster: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Cluster
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetCluster retrieves a cluster by ID
func (c *CloudClient) GetCluster(ctx context.Context, clusterID string) (*Cluster, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/clusters/"+clusterID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get cluster: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Cluster
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateCluster updates a cluster's mutable attributes
func (c *CloudClient) UpdateCluster(ctx context.Context, clusterID string, cluster *Cluster) (*Cluster, error) {
	body, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/clusters/"+clusterID, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update cluster: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result Cluster
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteCluster terminates a cluster
func (c *CloudClient) DeleteCluster(ctx context.Context, clusterID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/clusters/"+clusterID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete cluster: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// WaitForClusterReady polls until the cluster is in_service
func (c *CloudClient) WaitForClusterReady(ctx context.Context, clusterID string, timeout time.Duration) (*Cluster, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timeout waiting for cluster to be ready")
			}

			cluster, err := c.GetCluster(ctx, clusterID)
			if err != nil {
				return nil, err
			}

			if cluster.Status == "in_service" {
				return cluster, nil
			}

			if cluster.Status == "failed" || cluster.Status == "terminated" {
				return nil, fmt.Errorf("cluster entered %s state", cluster.Status)
			}
		}
	}
}

// CreateClusterConfigChange schedules a configuration change
func (c *CloudClient) CreateClusterConfigChange(ctx context.Context, change *ClusterConfigChange) (*ClusterConfigChange, error) {
	body, err := json.Marshal(change)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config change: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/clusters/"+change.ClusterID+"/configuration-changes", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create config change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create config change: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ClusterConfigChange
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetClusterConfigChange retrieves a configuration change
func (c *CloudClient) GetClusterConfigChange(ctx context.Context, clusterID, changeID string) (*ClusterConfigChange, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/clusters/"+clusterID+"/configuration-changes/"+changeID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get config change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get config change: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ClusterConfigChange
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteClusterConfigChange cancels a scheduled configuration change
func (c *CloudClient) DeleteClusterConfigChange(ctx context.Context, clusterID, changeID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/clusters/"+clusterID+"/configuration-changes/"+changeID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete config change: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete config change: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GenerateClusterAPIKeys generates new API keys for a cluster
func (c *CloudClient) GenerateClusterAPIKeys(ctx context.Context, clusterID string) (*ClusterAPIKeys, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/clusters/"+clusterID+"/api-keys", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to generate API keys: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result ClusterAPIKeys
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *CloudClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TYPESENSE-CLOUD-MANAGEMENT-API-KEY", c.apiKey)
}

// ListClusters retrieves all clusters
func (c *CloudClient) ListClusters(ctx context.Context) ([]Cluster, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/clusters", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list clusters: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result []Cluster
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
