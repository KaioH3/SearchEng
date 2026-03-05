package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/usuario/searcheng/api"
	"github.com/usuario/searcheng/config"
	"github.com/usuario/searcheng/engine"
	"golang.org/x/time/rate"
)

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

Search options:
  --providers=ddg,google,bing,brave   Comma-separated list of providers (default: all)
  --max-results=20                    Maximum number of results

Serve options:
  --port=8080                         Port to listen on
  --providers=ddg,google,bing,brave   Comma-separated list of providers (default: all)

Environment variables:
  BRAVE_API_KEY          Brave Search API key (free tier)
  SEARCHENG_PORT         Server port (default: 8080)
  SEARCHENG_TIMEOUT      Search timeout (default: 5s)
  SEARCHENG_MAX_RESULTS  Max results (default: 20)`)
}

func buildScrapingClient(cfg config.Config, reqsPerMinute float64) *http.Client {
	limiter := rate.NewLimiter(rate.Limit(reqsPerMinute/60.0), 1)
	transport := engine.NewRetryTransport(
		engine.NewRateLimitedTransport(http.DefaultTransport, limiter),
		cfg.MaxRetries, cfg.RetryBaseDelay,
	)
	return &http.Client{Timeout: cfg.Timeout, Transport: transport}
}

func buildEngine(cfg config.Config, providerFlag string) *engine.Engine {
	allProviders := map[string]engine.Provider{
		"ddg":    &engine.DuckDuckGo{Client: buildScrapingClient(cfg, 10)},
		"google": &engine.Google{Client: buildScrapingClient(cfg, 5)},
		"bing":   &engine.Bing{Client: buildScrapingClient(cfg, 10)},
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
		// Default: all available providers
		for _, p := range allProviders {
			providers = append(providers, p)
		}
	}

	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no providers available")
		os.Exit(1)
	}

	return &engine.Engine{
		Providers:  providers,
		Timeout:    cfg.Timeout,
		MaxResults: cfg.MaxResults,
	}
}

func cmdSearch(cfg config.Config, args []string) {
	var query string
	var providerFlag string
	var maxResults int

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--providers="):
			providerFlag = strings.TrimPrefix(arg, "--providers=")
		case strings.HasPrefix(arg, "--max-results="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--max-results="), "%d", &maxResults)
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

	eng := buildEngine(cfg, providerFlag)

	fmt.Printf("Searching for: %s\n", query)
	fmt.Printf("Providers: %d active\n", len(eng.Providers))
	fmt.Println(strings.Repeat("─", 60))

	resp := eng.Search(query, 1)

	if len(resp.Results) == 0 {
		fmt.Println("No results found.")
		return
	}

	for i, r := range resp.Results {
		fmt.Printf("\n%d. %s\n", i+1, r.Title)
		fmt.Printf("   %s\n", r.URL)
		if r.Snippet != "" {
			snippet := r.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
			fmt.Printf("   %s\n", snippet)
		}
		fmt.Printf("   [%s]\n", r.Source)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Found %d results in %s\n", len(resp.Results), resp.Duration.Round(time.Millisecond))
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

	eng := buildEngine(cfg, providerFlag)

	server := &api.Server{
		Engine: eng,
		Port:   cfg.Port,
	}

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
