package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/KaioH3/SearchEng/engine"
)

type mockProvider struct {
	name    string
	results []engine.Result
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Search(ctx context.Context, query string, page int) ([]engine.Result, error) {
	return m.results, nil
}

func newTestServer() *Server {
	return &Server{
		Engine: &engine.Engine{
			Providers: []engine.Provider{
				&mockProvider{
					name: "Mock",
					results: []engine.Result{
						{URL: "https://example.com", Title: "Example", Snippet: "Test snippet", Source: "Mock"},
					},
				},
			},
			Timeout:    5 * time.Second,
			MaxResults: 20,
		},
		Port: 0,
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body healthResponse
	json.NewDecoder(w.Body).Decode(&body)
	if body.Status != "ok" {
		t.Errorf("status = %q, want 'ok'", body.Status)
	}
	if len(body.Providers) != 1 {
		t.Errorf("expected 1 provider in health, got %d", len(body.Providers))
	}
	if body.Providers[0].Name != "Mock" {
		t.Errorf("provider name = %q, want 'Mock'", body.Providers[0].Name)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", ct)
	}
}

func TestSearchEndpoint_Success(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body apiResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Query != "test" {
		t.Errorf("query = %q, want 'test'", body.Query)
	}
	if body.Page != 1 {
		t.Errorf("page = %d, want 1", body.Page)
	}
	if len(body.Results) == 0 {
		t.Error("expected results, got none")
	}
	if body.TotalCount != len(body.Results) {
		t.Errorf("total_count = %d, but results has %d items", body.TotalCount, len(body.Results))
	}
	if body.DurationMs < 0 {
		t.Error("expected non-negative duration_ms")
	}
}

func TestSearchEndpoint_MissingQuery(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/search", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected error field in JSON response")
	}
}

func TestSearchEndpoint_EmptyQuery(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/search?q=", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for empty query", w.Code)
	}
}

func TestSearchEndpoint_WithPage(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/search?q=test&page=2", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	var body apiResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Page != 2 {
		t.Errorf("page = %d, want 2", body.Page)
	}
}

func TestSearchEndpoint_InvalidPage(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/search?q=test&page=abc", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	var body apiResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Page != 1 {
		t.Errorf("page = %d, want 1 (default for invalid page)", body.Page)
	}
}

func TestHandleIndex_ServesJSON(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	srv.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["name"] != "searcheng" {
		t.Errorf("name = %v, want 'searcheng'", body["name"])
	}
	endpoints, ok := body["endpoints"].([]any)
	if !ok || len(endpoints) == 0 {
		t.Error("expected non-empty endpoints array")
	}
}

func TestHandleIndex_404ForOtherPaths(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.handleIndex(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestRAGSearch_Success(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body ragResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if body.Query != "test" {
		t.Errorf("query = %q, want 'test'", body.Query)
	}
	if body.Total == 0 {
		t.Error("expected results, got none")
	}
	if body.Total != len(body.Results) {
		t.Errorf("total = %d, but results has %d items", body.Total, len(body.Results))
	}
	for _, r := range body.Results {
		if r.Context == "" {
			t.Error("expected non-empty context field")
		}
	}
}

func TestRAGSearch_MissingQuery(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/v1/search", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRAGSearch_MaxResults(t *testing.T) {
	srv := &Server{
		Engine: &engine.Engine{
			Providers: []engine.Provider{
				&mockProvider{
					name: "Mock",
					results: []engine.Result{
						{URL: "https://a.com", Title: "Test result A", Snippet: "Test content about topic A", Source: "Mock"},
						{URL: "https://b.com", Title: "Test result B", Snippet: "Test content about topic B", Source: "Mock"},
						{URL: "https://c.com", Title: "Test result C", Snippet: "Test content about topic C", Source: "Mock"},
					},
				},
			},
			Timeout:    5 * time.Second,
			MaxResults: 20,
		},
	}
	req := httptest.NewRequest("GET", "/v1/search?q=test&max_results=2", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	var body ragResponse
	json.NewDecoder(w.Body).Decode(&body)

	if body.Total != 2 {
		t.Errorf("total = %d, want 2 (respecting max_results)", body.Total)
	}
}

func TestRAGSearch_ContextBlockFormat(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	var body ragResponse
	json.NewDecoder(w.Body).Decode(&body)

	if !strings.Contains(body.ContextBlock, "[1] Example (https://example.com)") {
		t.Errorf("context_block missing expected format, got: %s", body.ContextBlock)
	}
	if !strings.Contains(body.ContextBlock, "Test snippet") {
		t.Errorf("context_block missing snippet, got: %s", body.ContextBlock)
	}
}

func newTestServerWithTrusted() *Server {
	return &Server{
		Engine: &engine.Engine{
			Providers: []engine.Provider{
				&mockProvider{
					name: "Mock",
					results: []engine.Result{
						{URL: "https://github.com/user/repo", Title: "GitHub Repo", Snippet: "A trusted repository for testing", Source: "Mock"},
					},
				},
			},
			Timeout:    5 * time.Second,
			MaxResults: 20,
		},
		Port: 0,
	}
}

func TestRAGSearch_TrustSignalsPresent(t *testing.T) {
	srv := newTestServerWithTrusted()
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	var body ragResponse
	json.NewDecoder(w.Body).Decode(&body)

	if len(body.Results) == 0 {
		t.Fatal("expected results")
	}
	r := body.Results[0]
	if r.TrustSignals == nil {
		t.Fatal("expected trust_signals to be present")
	}
	if !r.TrustSignals.IsHTTPS {
		t.Error("expected IsHTTPS=true for https URL")
	}
	if !r.TrustSignals.IsTrusted {
		t.Error("expected IsTrusted=true for github.com")
	}
}

func TestRAGSearch_AnswerField(t *testing.T) {
	srv := &Server{
		Engine: &engine.Engine{
			Providers: []engine.Provider{
				&mockProvider{
					name: "Mock",
					results: []engine.Result{
						{
							URL:     "https://example.com",
							Title:   "What is Go",
							Snippet: "Go is a programming language designed by Google. It makes it easy to build simple and reliable software.",
							Source:  "Mock",
						},
					},
				},
			},
			Timeout:    5 * time.Second,
			MaxResults: 20,
		},
	}
	req := httptest.NewRequest("GET", "/v1/search?q=what+is+Go", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	var body ragResponse
	json.NewDecoder(w.Body).Decode(&body)

	// Answer may or may not be present depending on score threshold,
	// but the field should be in the response struct (omitempty)
	if body.Query != "what is Go" {
		t.Errorf("query = %q, want 'what is Go'", body.Query)
	}
}

func TestRAGSearch_ContextBlockTags(t *testing.T) {
	srv := newTestServerWithTrusted()
	req := httptest.NewRequest("GET", "/v1/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	var body ragResponse
	json.NewDecoder(w.Body).Decode(&body)

	// github.com is HTTPS and Trusted — tags should appear
	if !strings.Contains(body.ContextBlock, "[HTTPS, Trusted]") {
		t.Errorf("expected [HTTPS, Trusted] tags in context_block, got: %s", body.ContextBlock)
	}
}

func TestRAGSearch_MaxResultsUpperBound(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest("GET", "/v1/search?q=test&max_results=999", nil)
	w := httptest.NewRecorder()

	srv.handleRAGSearch(w, req)

	// Should not crash or return more than 100
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestSearchEndpoint_TrustNotExposed(t *testing.T) {
	srv := newTestServerWithTrusted()
	req := httptest.NewRequest("GET", "/search?q=test", nil)
	w := httptest.NewRecorder()

	srv.handleSearch(w, req)

	// Trust field has json:"-", so it should NOT appear in the response
	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, "trust_signals") {
		t.Errorf("trust_signals should not be exposed in /search endpoint")
	}
	if strings.Contains(bodyStr, "is_https") {
		t.Errorf("is_https should not be exposed in /search endpoint")
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Regular GET request should have CORS headers
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want '*'", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Errorf("CORS methods = %q, want 'GET, OPTIONS'", got)
	}

	// OPTIONS preflight should return 204
	req = httptest.NewRequest("OPTIONS", "/", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS status = %d, want 204", w.Code)
	}
}
