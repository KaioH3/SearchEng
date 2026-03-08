# SearchEng

**A meta-search engine that combines results from multiple search providers into one ranked feed.**

SearchEng queries DuckDuckGo, Google, Bing, and Brave in parallel, deduplicates results, ranks them using information retrieval algorithms (RRF + BM25F), and extracts featured answers and factual claims from the results.

You can use it as a **CLI tool**, a **REST API with web UI**, or an **MCP server for AI agents**.

```
$ searcheng search "how does garbage collection work in Go"

Searching for: how does garbage collection work in Go
Providers: 3 active
────────────────────────────────────────────────────────────
  DuckDuckGo: 12 results
  Bing: 10 results
  Google: 8 results
────────────────────────────────────────────────────────────

💡 Go uses a concurrent, tri-color mark-and-sweep garbage collector that runs
   alongside application goroutines with sub-millisecond pause times.
────────────────────────────────────────────────────────────

1. A Guide to the Go Garbage Collector
   https://go.dev/doc/gc-guide
   Go's garbage collector is a non-generational concurrent collector...
   Score: 4.82 | Sources: DuckDuckGo, Google, Bing

2. Garbage Collection In Go
   https://en.wikipedia.org/wiki/Go_(programming_language)
   ...

Found 18 results in 1.2s
```

---

## Table of Contents

- [Why SearchEng?](#why-searcheng)
- [Getting Started](#getting-started)
  - [Requirements](#requirements)
  - [Installation](#installation)
  - [Quick Start](#quick-start)
- [Usage](#usage)
  - [CLI Search](#cli-search)
  - [Web UI & REST API](#web-ui--rest-api)
  - [MCP Server (AI Agents)](#mcp-server-ai-agents)
- [API Reference](#api-reference)
  - [Endpoints](#endpoints)
  - [GET /search](#get-search)
  - [GET /v1/search (RAG)](#get-v1search-rag)
  - [GET /health](#get-health)
- [Configuration](#configuration)
  - [Environment Variables](#environment-variables)
  - [Ranking Weights](#ranking-weights)
- [How It Works](#how-it-works)
  - [Search & Ranking Pipeline](#search--ranking-pipeline)
  - [Answer Extraction](#answer-extraction)
  - [Claim Extraction](#claim-extraction)
  - [Anti-Blocking](#anti-blocking)
- [Architecture](#architecture)
- [Development](#development)
- [License](#license)

---

## Why SearchEng?

Most search APIs are expensive, rate-limited, or require API keys. SearchEng scrapes free search engines directly and combines results into a single ranked feed.

**Use cases:**

| Use Case | How SearchEng Helps |
|---|---|
| **RAG pipelines** | `/v1/search` returns a pre-formatted `context_block` ready to inject into LLM prompts |
| **AI agents** | MCP server lets Claude, GPT, or any MCP-compatible agent search the web |
| **Research** | Cross-reference results across engines with trust signals and claim corroboration |
| **Self-hosted search** | No API keys required (Brave is optional for higher quality) |

---

## Getting Started

### Requirements

- **Go 1.24+** ([install Go](https://go.dev/dl/))
- No API keys required (Brave Search API is optional)

### Installation

**Option 1: `go install`**

```bash
go install github.com/KaioH3/SearchEng@latest
```

**Option 2: Build from source**

```bash
git clone https://github.com/KaioH3/SearchEng.git
cd SearchEng
go build -o searcheng .
```

Verify the installation:

```bash
./searcheng search "hello world"
```

### Quick Start

```bash
# Search from the terminal
./searcheng search "what is WebAssembly"

# Start the web UI + API server
./searcheng serve
# Open http://localhost:8080 in your browser

# Start the MCP server for AI agents
./searcheng mcp
```

---

## Usage

### CLI Search

```bash
# Basic search
searcheng search "golang concurrency patterns"

# Choose specific providers
searcheng search "rust vs go" --providers=ddg,brave

# Limit number of results
searcheng search "latest news" --max-results=5

# Disable result caching
searcheng search "breaking news" --no-cache

# Disable SafeSearch (NSFW filter)
searcheng search "query" --safe-search=false
```

**Available providers:** `ddg` (DuckDuckGo), `google`, `bing`, `brave` (requires API key)

### Web UI & REST API

```bash
# Start with default settings (port 8080, all providers)
searcheng serve

# Custom port and specific providers
searcheng serve --port=3000 --providers=ddg,google,bing
```

Open `http://localhost:8080` in your browser to use the web interface. It features:

- Search bar with suggested queries
- Featured answer box
- Result cards with favicons, source badges, and trust indicators
- Factual claims with confidence bars
- SafeSearch toggle
- Dark/light mode (automatic, follows your OS setting)
- Responsive design (works on mobile)
- Provider status in the footer

The same server also serves the JSON API. See [API Reference](#api-reference) for details.

### MCP Server (AI Agents)

The MCP (Model Context Protocol) server lets AI agents search the web through SearchEng.

```bash
searcheng mcp
```

**Claude Desktop** — add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "searcheng": {
      "command": "/path/to/searcheng",
      "args": ["mcp"]
    }
  }
}
```

**Claude Code** — add to `.mcp.json` in your project:

```json
{
  "mcpServers": {
    "searcheng": {
      "command": "/path/to/searcheng",
      "args": ["mcp"]
    }
  }
}
```

The MCP server exposes a `search` tool with parameters:

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `query` | string | Yes | — | Search query |
| `max_results` | number | No | `5` | Maximum results to return |
| `safe_search` | boolean | No | `true` | Enable/disable NSFW filter |

---

## API Reference

### Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web UI (browsers) or API info (JSON clients) |
| `GET` | `/search?q=...` | Standard search with ranked results |
| `GET` | `/v1/search?q=...` | RAG-optimized search with answer, claims, and trust signals |
| `GET` | `/health` | Provider health check |

The root `/` endpoint serves the web UI to browsers and returns JSON to API clients (based on the `Accept` header). To get JSON, send `Accept: application/json`:

```bash
curl -H "Accept: application/json" http://localhost:8080/
```

### GET /search

Standard search endpoint. Returns ranked results.

**Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `q` | string | *required* | Search query |
| `page` | integer | `1` | Page number |
| `safe_search` | string | `true` | Set to `false` or `0` to disable |

**Example:**

```bash
curl "http://localhost:8080/search?q=golang+error+handling&page=1"
```

```json
{
  "query": "golang error handling",
  "page": 1,
  "results": [
    {
      "title": "Error handling in Go",
      "url": "https://go.dev/blog/error-handling",
      "snippet": "Go uses explicit error returns instead of exceptions...",
      "source": "DuckDuckGo, Google"
    }
  ],
  "total_count": 18,
  "duration_ms": 1200,
  "providers": [
    { "name": "DuckDuckGo", "success": true, "count": 10 },
    { "name": "Google", "success": true, "count": 8 },
    { "name": "Bing", "success": true, "count": 10 }
  ]
}
```

### GET /v1/search (RAG)

RAG-optimized endpoint. Returns everything from `/search` plus: `answer`, `claims`, `trust_signals`, `context_block`, and relevance `score`.

This is the endpoint the web UI uses.

**Parameters:**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `q` | string | *required* | Search query |
| `max_results` | integer | `5` | Max results (capped at 100) |
| `safe_search` | string | `true` | Set to `false` or `0` to disable |

**Example:**

```bash
curl "http://localhost:8080/v1/search?q=golang+error+handling&max_results=3"
```

```json
{
  "query": "golang error handling",
  "results": [
    {
      "title": "Error handling in Go",
      "url": "https://go.dev/blog/error-handling",
      "snippet": "Go uses explicit error returns instead of exceptions...",
      "source": "DuckDuckGo, Google",
      "score": 4.12,
      "context": "Error handling in Go\nGo uses explicit error returns...",
      "trust_signals": {
        "is_https": true,
        "tld": ".dev",
        "tld_category": "developer",
        "is_trusted": true,
        "trusted_domain": "go.dev",
        "source_count": 2
      }
    }
  ],
  "answer": "Go uses explicit error returns instead of exceptions, making error handling a first-class part of the language.",
  "claims": [
    {
      "text": "Go 1.13 introduced the errors.Is and errors.As functions",
      "sources": ["DuckDuckGo", "Google"],
      "corroboration": 2,
      "confidence": 0.8
    }
  ],
  "context_block": "[1] Error handling in Go (https://go.dev/blog/error-handling) [HTTPS, Trusted]\nGo uses explicit error returns...",
  "total": 3,
  "duration_ms": 1200
}
```

**Using `context_block` in a RAG prompt:**

```
Based on the following search results, answer the user's question.

{context_block from /v1/search}

Question: {user's question}
```

### GET /health

Returns the status of all configured providers.

```bash
curl http://localhost:8080/health
```

```json
{
  "status": "ok",
  "providers": [
    { "name": "DuckDuckGo", "available": true },
    { "name": "Google", "available": true },
    { "name": "Bing", "available": true },
    { "name": "Brave", "available": true }
  ]
}
```

---

## Configuration

### Environment Variables

All configuration is done via environment variables. No config files needed.

**Core settings:**

| Variable | Default | Description |
|---|---|---|
| `SEARCHENG_PORT` | `8080` | Server port for `serve` command |
| `SEARCHENG_TIMEOUT` | `5s` | Timeout per search request |
| `SEARCHENG_MAX_RESULTS` | `20` | Maximum results returned |
| `SEARCHENG_CACHE_TTL` | `1h` | How long results are cached (`0` to disable) |
| `SEARCHENG_SAFE_SEARCH` | `true` | NSFW content filtering |

**API keys:**

| Variable | Default | Description |
|---|---|---|
| `BRAVE_API_KEY` | — | Brave Search API key ([free tier: 2000 queries/month](https://brave.com/search/api/)) |

**Anti-blocking:**

| Variable | Default | Description |
|---|---|---|
| `SEARCHENG_MAX_RETRIES` | `2` | Max retries on 429/5xx errors |
| `SEARCHENG_RETRY_DELAY` | `500ms` | Base delay for exponential backoff |
| `SEARCHENG_GOOGLE_RPM` | `1` | Google: max requests per minute |
| `SEARCHENG_DDG_RPM` | `10` | DuckDuckGo: max requests per minute |
| `SEARCHENG_BING_RPM` | `10` | Bing: max requests per minute |

### Ranking Weights

Fine-tune how results are scored. Higher values = more influence on final ranking.

| Variable | Default | Description |
|---|---|---|
| `SEARCHENG_RANK_POSITION_W` | `0.4` | Weight for position-based RRF score |
| `SEARCHENG_RANK_BM25_W` | `0.3` | Weight for BM25F text relevance |
| `SEARCHENG_RANK_MULTISOURCE_W` | `0.2` | Bonus for results found in multiple engines |
| `SEARCHENG_RANK_SNIPPET_W` | `0.1` | Weight for snippet quality |
| `SEARCHENG_RANK_TRUSTED_DOMAIN_BONUS` | `0.5` | Bonus for trusted domains (GitHub, Wikipedia, etc.) |
| `SEARCHENG_RANK_TLD_W` | `0.3` | Weight for TLD category score |
| `SEARCHENG_RANK_HTTPS_BONUS` | `0.1` | Bonus for HTTPS results |

---

## How It Works

### Search & Ranking Pipeline

When you make a search, here's what happens:

```
1. Query received
       │
2. Check cache ──→ cache hit? return cached results
       │ (miss)
3. Fan out to all providers in parallel (goroutines)
       │
4. Collect results (with timeout)
       │
5. Deduplicate by URL (merge sources)
       │
6. Score each result:
       │   ├── RRF: Reciprocal Rank Fusion across providers
       │   ├── BM25F: text relevance (title 3x, URL 2x, snippet 1x)
       │   ├── Multi-source bonus (found in 2+ engines)
       │   ├── Trust signals (HTTPS, TLD, trusted domains)
       │   ├── Language penalty (CJK mismatch)
       │   └── Coverage penalty (<15% query terms matched)
       │
7. Sort by score, discard negatives
       │
8. Extract answer (best snippet sentence)
       │
9. Extract claims (cross-source corroboration)
       │
10. Cache results, return response
```

**Scoring formula:**

```
score = PositionW    * RRF * 100
      + BM25W        * BM25F(title*3, snippet*1, url*2)
      + MultiSourceW * (sourceCount - 1) * 10
      + SnippetW     * snippetQuality
      + trustedDomainBonus
      + tldScore           (.edu/.gov: +1.0, .org: +0.5, .xyz/.click: -0.5)
      + httpsBonus
      + languagePenalty    (CJK mismatch: -3.0)
      + coveragePenalty    (low query coverage: -2.0)
```

### Answer Extraction

SearchEng picks the best sentence from the top results as a "featured answer":

1. Split top snippets into sentences
2. Score each sentence by: query term overlap, presence of a definition pattern ("is a", "refers to"), length, and source trust
3. The highest-scoring sentence above a threshold becomes the featured answer
4. SafeSearch filters out NSFW content from answers

### Claim Extraction

Factual claims are statements that can be verified. SearchEng extracts them automatically:

1. Identify sentences containing numbers, dates, comparisons, or attribution ("according to...")
2. Group similar claims across sources using Jaccard similarity (threshold > 0.4)
3. Calculate confidence based on: corroboration count, trusted source bonus, and claim strength
4. Claims found in multiple independent sources get higher confidence

### Anti-Blocking

Scraping search engines requires careful request management. SearchEng uses multiple layers:

| Layer | What It Does |
|---|---|
| **Rate limiting** | Per-provider rate limiter (Google: 1 req/min, DDG/Bing: 10 req/min) |
| **Jittered delays** | Random 0.5-2s delay before each request to look natural |
| **Exponential backoff** | Automatic retry on 429/5xx with increasing wait times |
| **Browser mimicry** | Rotating User-Agent strings with matching `Sec-CH-UA` and `Sec-Fetch-*` headers |
| **Cookie jars** | Per-provider cookie persistence (maintains session context) |
| **CAPTCHA detection** | Google: detects `/sorry/` redirects and CAPTCHA pages, triggers 5-min cooldown |
| **Consent bypass** | Google: sends consent cookies to skip EU consent wall |

Google is the most aggressive at blocking, so it uses a stricter rate limit (1 RPM) and no automatic retries.

---

## Architecture

```
                      ┌─────────────────────┐
                      │      main.go        │
                      │  search │ serve│ mcp│
                      └───┬─────┼──────┼────┘
                          │     │      │
                     ┌────┘     │      └────┐
                     ▼          ▼           ▼
                ┌─────────┐ ┌────────┐ ┌────────┐
                │   CLI   │ │  REST  │ │  MCP   │
                │ stdout  │ │ API +  │ │ stdio  │
                │         │ │ Web UI │ │        │
                └────┬────┘ └───┬────┘ └───┬────┘
                     │          │          │
                     └──────────┼──────────┘
                                │
                        ┌───────▼───────┐
                        │    Engine     │
                        │  - parallel   │
                        │  - rank/merge │
                        │  - answer     │
                        │  - claims     │
                        │  - cache      │
                        └──┬──┬──┬──┬───┘
                           │  │  │  │
                    ┌──────┘  │  │  └──────┐
                    ▼         ▼  ▼         ▼
                ┌──────┐ ┌──────┐ ┌────┐ ┌─────┐
                │ DDG  │ │Google│ │Bing│ │Brave│
                │scrape│ │scrape│ │scrp│ │ API │
                └──────┘ └──────┘ └────┘ └─────┘
```

**Project structure:**

```
SearchEng/
├── main.go                 Entry point: CLI, serve, and MCP commands
├── web/
│   └── index.html          Web UI (Alpine.js + Pico CSS, served via go:embed)
├── api/
│   ├── server.go           REST API server with CORS and content negotiation
│   └── server_test.go      API tests
├── engine/
│   ├── engine.go           Search aggregator (parallel query, RRF + BM25F ranking)
│   ├── provider.go         Provider interface
│   ├── result.go           Result, Claim, TrustSignals types
│   ├── duckduckgo.go       DuckDuckGo HTML scraper
│   ├── google.go           Google HTML scraper (CAPTCHA detection + cooldown)
│   ├── bing.go             Bing HTML scraper (tracking URL decoder)
│   ├── brave.go            Brave Search API client
│   ├── answer.go           Featured answer extraction
│   ├── claims.go           Factual claim extraction + corroboration
│   ├── cache.go            Thread-safe in-memory cache with TTL
│   ├── stopwords.go        English + Portuguese stopword lists
│   ├── httpclient.go       HTTP transports (retry, rate-limit, jitter, browser headers)
│   └── *_test.go           Tests for each component
├── mcp/
│   ├── server.go           MCP server (JSON-RPC 2.0 over stdio)
│   └── server_test.go      MCP tests
└── config/
    └── config.go           Environment-based configuration
```

---

## Development

```bash
# Run all tests (with race detector)
go test ./... -race -count=1

# Run tests with verbose output
go test ./... -race -v

# Build the binary
go build -o searcheng .

# Run static analysis
go vet ./...

# Run the server in development
go run . serve
```

**Test coverage:** 183 tests across 4 packages, all passing with `-race`.

---

## License

[MIT](LICENSE)
