//go:build e2e

package chinooktest

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockOpenAI is an in-process OpenAI-compatible HTTP server bound to all
// interfaces so a Typesense container can reach it via host.docker.internal.
// Records every incoming request for assertions.
type MockOpenAI struct {
	URL   string // for use from inside containers (host.docker.internal:<port>)
	Local string // for use from the host process

	server *httptest.Server

	mu       sync.Mutex
	requests []string
}

// StartMockOpenAI launches the mock server on a free port. The server is
// stopped on test cleanup.
func StartMockOpenAI(t *testing.T) *MockOpenAI {
	t.Helper()

	m := &MockOpenAI{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		m.recordRequest(r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockChatCompletion())
	})

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("StartMockOpenAI: listen: %v", err)
	}

	srv := httptest.NewUnstartedServer(mux)
	srv.Listener = listener
	srv.Start()

	t.Cleanup(srv.Close)

	port := listener.Addr().(*net.TCPAddr).Port
	m.URL = fmt.Sprintf("http://host.docker.internal:%d", port)
	m.Local = fmt.Sprintf("http://127.0.0.1:%d", port)
	m.server = srv
	return m
}

// Requests returns a snapshot of recorded requests in "<METHOD> <path>" form.
func (m *MockOpenAI) Requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.requests))
	copy(out, m.requests)
	return out
}

func (m *MockOpenAI) recordRequest(r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, r.Method+" "+r.URL.Path)
}

func mockChatCompletion() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-mock",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "mock",
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": `{"q":"*"}`},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	}
}
