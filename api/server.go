package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/KaioH3/SearchEng/engine"
)

// Server is the REST API server for the search engine.
type Server struct {
	Engine    *engine.Engine
	Port      int
	IndexHTML []byte
}

// NewHTTPServer creates an *http.Server with production timeouts and CORS.
func (s *Server) NewHTTPServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/search", s.handleRAGSearch)

	addr := fmt.Sprintf(":%d", s.Port)
	slog.Info("server starting", "url", fmt.Sprintf("http://localhost:%d", s.Port))

	return &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	accept := r.Header.Get("Accept")
	wantsJSON := strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html")
	if !wantsJSON && len(s.IndexHTML) > 0 {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(s.IndexHTML)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":      "searcheng",
		"version":   "0.1.0",
		"endpoints": []string{"/search", "/v1/search", "/health"},
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONError(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	var opts []engine.SearchOptions
	if ss := r.URL.Query().Get("safe_search"); ss == "false" || ss == "0" {
		f := false
		opts = append(opts, engine.SearchOptions{SafeSearch: &f})
	}

	resp := s.Engine.Search(r.Context(), query, page, opts...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse{
		Query:      resp.Query,
		Page:       resp.Page,
		Results:    resp.Results,
		TotalCount: len(resp.Results),
		DurationMs: resp.Duration.Milliseconds(),
		Providers:  resp.Providers,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	providers := make([]providerHealth, 0, len(s.Engine.Providers))
	for _, p := range s.Engine.Providers {
		providers = append(providers, providerHealth{Name: p.Name(), Available: true})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse{
		Status:    "ok",
		Providers: providers,
	})
}

func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

type apiResponse struct {
	Query      string                  `json:"query"`
	Page       int                     `json:"page"`
	Results    []engine.Result         `json:"results"`
	TotalCount int                     `json:"total_count"`
	DurationMs int64                   `json:"duration_ms"`
	Providers  []engine.ProviderStatus `json:"providers,omitempty"`
}

type healthResponse struct {
	Status    string           `json:"status"`
	Providers []providerHealth `json:"providers"`
}

type providerHealth struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
}

type ragResult struct {
	Title        string              `json:"title"`
	URL          string              `json:"url"`
	Snippet      string              `json:"snippet"`
	Source       string              `json:"source"`
	Score        float64             `json:"score"`
	Context      string              `json:"context"`
	TrustSignals *engine.TrustSignals `json:"trust_signals,omitempty"`
}

type ragResponse struct {
	Query        string         `json:"query"`
	Results      []ragResult    `json:"results"`
	Answer       string         `json:"answer,omitempty"`
	Claims       []engine.Claim `json:"claims,omitempty"`
	ContextBlock string         `json:"context_block"`
	Total        int            `json:"total"`
	DurationMs   int64          `json:"duration_ms"`
}

func (s *Server) handleRAGSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSONError(w, "missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	maxResults := 5
	if m := r.URL.Query().Get("max_results"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			if parsed > 100 {
				parsed = 100
			}
			maxResults = parsed
		}
	}

	var opts []engine.SearchOptions
	if ss := r.URL.Query().Get("safe_search"); ss == "false" || ss == "0" {
		f := false
		opts = append(opts, engine.SearchOptions{SafeSearch: &f})
	}

	resp := s.Engine.Search(r.Context(), query, 1, opts...)

	results := resp.Results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	ragResults := make([]ragResult, len(results))
	var ctxParts []string

	if resp.Answer != "" {
		ctxParts = append(ctxParts, fmt.Sprintf("Answer: %s", resp.Answer))
	}

	for i, r := range results {
		ragResults[i] = ragResult{
			Title:        r.Title,
			URL:          r.URL,
			Snippet:      r.Snippet,
			Source:       r.Source,
			Score:        r.Score,
			Context:      r.Title + "\n" + r.Snippet,
			TrustSignals: r.Trust,
		}

		// Build enriched context line
		var tags []string
		if r.Trust != nil {
			if r.Trust.IsHTTPS {
				tags = append(tags, "HTTPS")
			}
			if r.Trust.IsTrusted {
				tags = append(tags, "Trusted")
			}
			if r.Trust.TLDCategory != "" && r.Trust.TLDCategory != "other" && r.Trust.TLDCategory != "commercial" {
				tags = append(tags, r.Trust.TLDCategory)
			}
		}
		tagStr := ""
		if len(tags) > 0 {
			tagStr = " [" + strings.Join(tags, ", ") + "]"
		}
		ctxParts = append(ctxParts, fmt.Sprintf("[%d] %s (%s)%s\n%s", i+1, r.Title, r.URL, tagStr, r.Snippet))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ragResponse{
		Query:        query,
		Results:      ragResults,
		Answer:       resp.Answer,
		Claims:       resp.Claims,
		ContextBlock: strings.Join(ctxParts, "\n\n"),
		Total:        len(ragResults),
		DurationMs:   resp.Duration.Milliseconds(),
	})
}
