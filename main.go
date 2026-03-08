package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/KaioH3/SearchEng/api"
	"github.com/KaioH3/SearchEng/config"
	"github.com/KaioH3/SearchEng/engine"
	"github.com/KaioH3/SearchEng/mcp"
	"golang.org/x/time/rate"
)

//go:embed web/index.html
var indexHTML []byte

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.Load()

	switch os.Args[1] {
	case "search":
		cmdSearch(cfg, os.Args[2:])
	case "serve":
		cmdServe(cfg, os.Args[2:])
	case "mcp":
		cmdMCP(cfg, os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: searcheng <command> [options]

Commands:
  search "query"    Search and display results in terminal
  serve             Start the REST API server
  mcp               Start MCP (Model Context Protocol) server over stdio

Search options:
  --providers=ddg,google,bing,brave   Comma-separated list of providers (default: all)
  --max-results=20                    Maximum number of results
  --no-cache                          Disable result caching
  --safe-search=false                 Disable NSFW content filtering (default: true)

Serve options:
  --port=8080                         Port to listen on
  --providers=ddg,google,bing,brave   Comma-separated list of providers (default: all)

Environment variables:
  BRAVE_API_KEY          Brave Search API key (free tier)
  SEARCHENG_PORT         Server port (default: 8080)
  SEARCHENG_TIMEOUT      Search timeout (default: 5s)
  SEARCHENG_MAX_RESULTS  Max results (default: 20)
  SEARCHENG_MAX_RETRIES  Max retries on 429/5xx (default: 2)
  SEARCHENG_RETRY_DELAY  Base delay for retry backoff (default: 500ms)
  SEARCHENG_CACHE_TTL    Cache time-to-live (default: 1h, 0 to disable)
  SEARCHENG_SAFE_SEARCH  NSFW content filter (default: true)
  SEARCHENG_GOOGLE_RPM   Google requests per minute (default: 1)
  SEARCHENG_DDG_RPM      DuckDuckGo requests per minute (default: 10)
  SEARCHENG_BING_RPM     Bing requests per minute (default: 10)`)
}

func buildScrapingClient(cfg config.Config, reqsPerMinute float64) *http.Client {
	limiter := rate.NewLimiter(rate.Limit(reqsPerMinute/60.0), 1)
	transport := engine.NewRetryTransport(
		engine.NewRateLimitedTransport(
			engine.NewJitteredTransport(http.DefaultTransport, 500*time.Millisecond, 2*time.Second),
			limiter,
		),
		cfg.MaxRetries, cfg.RetryBaseDelay,
	)
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: cfg.Timeout, Transport: transport, Jar: jar}
}

func buildScrapingClientNoRetry(cfg config.Config, reqsPerMinute float64) *http.Client {
	limiter := rate.NewLimiter(rate.Limit(reqsPerMinute/60.0), 1)
	transport := engine.NewRateLimitedTransport(
		engine.NewJitteredTransport(http.DefaultTransport, 500*time.Millisecond, 2*time.Second),
		limiter,
	)
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: cfg.Timeout, Transport: transport, Jar: jar}
}

func buildEngine(cfg config.Config, providerFlag string, useCache bool, safeSearch ...bool) *engine.Engine {
	allProviders := map[string]engine.Provider{
		"ddg":    &engine.DuckDuckGo{Client: buildScrapingClient(cfg, cfg.DDGRPM)},
		"google": &engine.Google{Client: buildScrapingClientNoRetry(cfg, cfg.GoogleRPM)},
		"bing":   &engine.Bing{Client: buildScrapingClient(cfg, cfg.BingRPM)},
	}

	if cfg.BraveAPIKey != "" {
		braveClient := &http.Client{Timeout: cfg.Timeout}
		allProviders["brave"] = &engine.Brave{APIKey: cfg.BraveAPIKey, Client: braveClient}
	}

	var providers []engine.Provider

	if providerFlag != "" {
		for _, name := range strings.Split(providerFlag, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if p, ok := allProviders[name]; ok {
				providers = append(providers, p)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: unknown provider %q\n", name)
			}
		}
	} else {
		for _, p := range allProviders {
			providers = append(providers, p)
		}
	}

	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers available")
		os.Exit(1)
	}

	safe := cfg.SafeSearch
	if len(safeSearch) > 0 {
		safe = safeSearch[0]
	}

	eng := &engine.Engine{
		Providers:  providers,
		Timeout:    cfg.Timeout,
		MaxResults: cfg.MaxResults,
		SafeSearch: safe,
		Ranking: engine.RankingWeights{
			PositionW:          cfg.Ranking.PositionW,
			BM25W:              cfg.Ranking.BM25W,
			MultiSourceW:       cfg.Ranking.MultiSourceW,
			SnippetW:           cfg.Ranking.SnippetW,
			TrustedDomainBonus: cfg.Ranking.TrustedDomainBonus,
			TLDWeight:          cfg.Ranking.TLDWeight,
			HTTPSBonus:         cfg.Ranking.HTTPSBonus,
		},
	}

	if useCache && cfg.CacheTTL > 0 {
		eng.Cache = engine.NewCache(cfg.CacheTTL)
	}

	return eng
}

func cmdSearch(cfg config.Config, args []string) {
	var query string
	var providerFlag string
	var maxResults int
	noCache := false
	safeSearchSet := false
	safeSearchVal := cfg.SafeSearch

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--providers="):
			providerFlag = strings.TrimPrefix(arg, "--providers=")
		case strings.HasPrefix(arg, "--max-results="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-results="), "%d", &maxResults)
		case arg == "--no-cache":
			noCache = true
		case strings.HasPrefix(arg, "--safe-search="):
			safeSearchSet = true
			v := strings.TrimPrefix(arg, "--safe-search=")
			safeSearchVal = v != "false" && v != "0"
		default:
			if query == "" {
				query = arg
			} else {
				query += " " + arg
			}
		}
	}

	if query == "" {
		fmt.Fprintln(os.Stderr, "Error: no query provided")
		fmt.Fprintln(os.Stderr, `Usage: searcheng search "your query"`)
		os.Exit(1)
	}

	if maxResults > 0 {
		cfg.MaxResults = maxResults
	}

	var eng *engine.Engine
	if safeSearchSet {
		eng = buildEngine(cfg, providerFlag, !noCache, safeSearchVal)
	} else {
		eng = buildEngine(cfg, providerFlag, !noCache)
	}

	fmt.Printf("Searching for: %s\n", query)
	fmt.Printf("Providers: %d active\n", len(eng.Providers))
	fmt.Println(strings.Repeat("─", 60))

	resp := eng.Search(context.Background(), query, 1)

	// Show provider status
	for _, ps := range resp.Providers {
		if ps.Success {
			fmt.Printf("  %s: %d results\n", ps.Name, ps.Count)
		} else {
			fmt.Printf("  %s: %s\n", ps.Name, ps.Error)
		}
	}
	fmt.Println(strings.Repeat("─", 60))

	if resp.Answer != "" {
		fmt.Printf("\n%s %s\n", "\U0001f4a1", resp.Answer)
		fmt.Println(strings.Repeat("\u2500", 60))
	}

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		return
	}

	for i, r := range resp.Results {
		fmt.Printf("\n%d. %s\n", i+1, r.Title)
		fmt.Printf("   %s\n", r.URL)
		if r.Snippet != "" {
			snippet := r.Snippet
			if runes := []rune(snippet); len(runes) > 200 {
				snippet = string(runes[:200]) + "..."
			}
			fmt.Printf("   %s\n", snippet)
		}
		fmt.Printf("   Score: %.2f | Sources: %s\n", r.Score, r.Source)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d results in %s\n", len(resp.Results), resp.Duration.Round(time.Millisecond))
}

func cmdMCP(cfg config.Config, args []string) {
	var providerFlag string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--providers=") {
			providerFlag = strings.TrimPrefix(arg, "--providers=")
		}
	}

	eng := buildEngine(cfg, providerFlag, true)
	srv := mcp.NewServer(eng)
	if err := srv.Run(os.Stdin, os.Stdout); err != nil {
		slog.Error("mcp server error", "error", err)
		os.Exit(1)
	}
}

func cmdServe(cfg config.Config, args []string) {
	var providerFlag string

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--port="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--port="), "%d", &cfg.Port)
		case strings.HasPrefix(arg, "--providers="):
			providerFlag = strings.TrimPrefix(arg, "--providers=")
		}
	}

	eng := buildEngine(cfg, providerFlag, true)

	s := &api.Server{
		Engine:    eng,
		Port:      cfg.Port,
		IndexHTML: indexHTML,
	}

	httpServer := s.NewHTTPServer()

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	} else {
		slog.Info("server stopped gracefully")
	}
}
