package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/usuario/searcheng/engine"
)

// Server is the REST API server for the search engine.
type Server struct {
	Engine *engine.Engine
	Port   int
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.Port)
	fmt.Printf("Starting server on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"missing query parameter 'q'"}`, http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	resp := s.Engine.Search(query, page)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse{
		Query:      resp.Query,
		Page:       resp.Page,
		Results:    resp.Results,
		TotalCount: len(resp.Results),
		DurationMs: resp.Duration.Milliseconds(),
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type apiResponse struct {
	Query      string          `json:"query"`
	Page       int             `json:"page"`
	Results    []engine.Result `json:"results"`
	TotalCount int             `json:"total_count"`
	DurationMs int64           `json:"duration_ms"`
}
