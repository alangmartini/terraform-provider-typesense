//go:build e2e

package chinooktest

import (
	"context"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
)

// TestMockOpenAISmoke proves the mock is reachable from a Typesense
// container via host.docker.internal: creates an nl_search_model with
// api_url pointing at the mock and asserts the mock recorded a request.
func TestMockOpenAISmoke(t *testing.T) {
	cluster := StartCluster(t, "30.1")
	mock := StartMockOpenAI(t)

	model := &client.NLSearchModel{
		ID:        "mock-openai-smoke",
		ModelName: "openai/test",
		APIKey:    "fake-key",
		APIURL:    mock.URL,
		MaxBytes:  16000,
	}

	if _, err := cluster.Client().CreateNLSearchModel(context.Background(), model); err != nil {
		t.Fatalf("CreateNLSearchModel: %v", err)
	}

	if got := mock.Requests(); len(got) == 0 {
		t.Errorf("mock received no requests; expected at least one POST from Typesense")
	}
}
