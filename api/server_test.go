package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/usuario/searcheng/engine"
)

type mockProvider struct {
	name    string
	results []engine.Result
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Search(query string, page int) ([]engine.Result, error) {
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

	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want 'ok'", body["status"])
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

	// Should default to page 1
	if body.Page != 1 {
		t.Errorf("page = %d, want 1 (default for invalid page)", body.Page)
	}
}
