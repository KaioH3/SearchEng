# searcheng

A meta-search engine written in Go that queries multiple providers in parallel, deduplicates and ranks results using RRF + BM25F scoring, extracts answers and factual claims, and exposes everything via CLI, REST API, and MCP (Model Context Protocol) for AI agent integration.

```
searcheng search "how does garbage collection work in Go"

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

## Why

Most search APIs are expensive, rate-limited, or require API keys. SearchEng scrapes free search engines directly and combines their results into a single ranked feed — useful for:

- **RAG pipelines** — plug real-time web context into LLM prompts
- **AI agents** — MCP server lets Claude, GPT, or any MCP-compatible agent search the web
- **Research tools** — cross-reference results across engines with trust signals and claim extraction
- **Self-hosted search** — no API keys needed (Brave optional for higher quality)

## Features

**Search & Ranking**
- Parallel search across DuckDuckGo, Google, Bing, and Brave (API)
- Reciprocal Rank Fusion (RRF) + BM25F scoring with field weights (title 3x, URL 2x)
- Multi-source boost — results found in multiple engines rank higher
- Trust signals — HTTPS, TLD scoring (.edu/.gov > .com > .xyz), trusted domain detection
- Query coverage penalty — filters irrelevant results automatically
- Language detection — Portuguese/English query routing with CJK mismatch penalty

**Content Intelligence**
- Answer extraction — picks the best sentence from top results as a featured answer
- Claim extraction — identifies factual assertions, corroborates across sources via Jaccard similarity
- SafeSearch — NSFW domain blocklist + explicit content detection with word-boundary matching

**Infrastructure**
- MCP server (JSON-RPC 2.0 over stdio) — integrates with Claude Desktop, Cursor, and any MCP client
- REST API with RAG-optimized endpoint (`/v1/search`)
- In-memory cache with TTL and SafeSearch-aware keys
- Anti-blocking: retry with exponential backoff, per-provider rate limiting, jittered delays, cookie jars, rotating user agents with matching `Sec-CH-UA` headers
- Graceful shutdown with signal handling

**Quality**
- 107 tests with `-race` flag — concurrent safety verified
- Zero goroutine leaks — cache cleanup is stoppable
- `go vet` clean

## Install

```bash
go install github.com/KaioH3/SearchEng@latest
```

Or build from source:

```bash
git clone https://github.com/KaioH3/SearchEng.git
cd searcheng
go build -o searcheng .
```

## Usage

### CLI Search

```bash
# Basic search
searcheng search "golang concurrency patterns"

# Select providers and limit results
searcheng search "rust vs go" --providers=ddg,brave --max-results=10

# Disable cache
searcheng search "breaking news" --no-cache

# Disable SafeSearch
searcheng search "query" --safe-search=false
```

### REST API

```bash
# Start the server
searcheng serve

# Custom port and providers
searcheng serve --port=3000 --providers=ddg,google,bing
```

### MCP Server (AI Agent Integration)

```bash
# Start MCP server over stdio (for Claude Desktop, Cursor, etc.)
searcheng mcp
```

Add to your Claude Desktop config (`claude_desktop_config.json`):

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

The MCP server exposes a `search` tool with parameters: `query` (required), `max_results`, `safe_search`.

## API Endpoints

| Endpoint | Description |
|---|---|
| `GET /search?q=query&page=1` | Standard search with ranked results |
| `GET /v1/search?q=query&max_results=5` | RAG-optimized — includes `context_block`, `answer`, `claims`, and `trust_signals` |
| `GET /health` | Health check with provider status |
| `GET /` | API info |

Both `/search` and `/v1/search` accept `safe_search=false` to disable NSFW filtering.

### RAG Integration

The `/v1/search` endpoint returns a pre-formatted `context_block` ready to inject into LLM prompts:

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
      "text": "Go 1.13 introduced the errors.Is and errors.As functions for error inspection",
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

## How Ranking Works

Each result is scored by combining multiple signals:

```
score = PositionW × RRF × 100
      + BM25W × BM25F(title×3, snippet×1, url×2)
      + MultiSourceW × (sourceCount - 1) × 10
      + SnippetW × snippetQuality
      + trustedDomainBonus
      + tldScore
      + httpsBonus
      + languagePenalty
      + queryCoveragePenalty
```

| Signal | What it does |
|---|---|
| **RRF** | Reciprocal Rank Fusion — rewards results ranked high across multiple providers |
| **BM25F** | Field-weighted BM25 with title boost (3x), URL boost (2x), IDF floor to prevent negative scores |
| **Multi-source** | Results found in 2+ engines get a bonus |
| **Trusted domains** | wikipedia.org, github.com, stackoverflow.com, go.dev, etc. get a fixed bonus |
| **TLD scoring** | .edu/.gov = +1.0, .org = +0.5, .com = 0, .xyz/.click = -0.5 |
| **HTTPS** | HTTPS results get a small bonus |
| **Language penalty** | CJK-heavy snippets with non-CJK queries get -3.0 |
| **Coverage penalty** | Results matching <15% of query terms get -2.0 |

Results with score < 0 are discarded.

## Anti-Blocking Strategy

Scraping search engines requires care. SearchEng uses a layered approach:

| Layer | Mechanism |
|---|---|
| **Rate limiting** | Per-provider `x-time/rate` limiter (Google: 1 RPM, DDG/Bing: 10 RPM) |
| **Jittered delays** | Random 0.5–2s delay before each request |
| **Retry transport** | Exponential backoff on 429/5xx with jitter |
| **Browser mimicry** | Rotating User-Agents with matching `Sec-CH-UA`, `Sec-Fetch-*` headers |
| **Cookie jars** | Per-provider cookie persistence across requests |
| **CAPTCHA detection** | Google: detects `/sorry/` redirects and CAPTCHA forms, triggers 5-min cooldown |
| **Consent bypass** | Google: sends consent cookies to skip the EU consent wall |
| **No retry on Google** | Google gets rate-limited transport without retry to avoid trigger-happy blocking |

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|---|---|---|
| `BRAVE_API_KEY` | — | Brave Search API key ([free tier](https://brave.com/search/api/): 2000 queries/month) |
| `SEARCHENG_PORT` | `8080` | Server port |
| `SEARCHENG_TIMEOUT` | `5s` | Per-search timeout |
| `SEARCHENG_MAX_RESULTS` | `20` | Maximum results |
| `SEARCHENG_MAX_RETRIES` | `2` | Retries on 429/5xx |
| `SEARCHENG_RETRY_DELAY` | `500ms` | Base retry delay |
| `SEARCHENG_CACHE_TTL` | `1h` | Cache TTL (`0` to disable) |
| `SEARCHENG_SAFE_SEARCH` | `true` | NSFW content filtering |
| `SEARCHENG_GOOGLE_RPM` | `1` | Google requests/minute |
| `SEARCHENG_DDG_RPM` | `10` | DuckDuckGo requests/minute |
| `SEARCHENG_BING_RPM` | `10` | Bing requests/minute |

Ranking weights are also configurable: `SEARCHENG_RANK_POSITION_W`, `SEARCHENG_RANK_BM25_W`, `SEARCHENG_RANK_MULTISOURCE_W`, `SEARCHENG_RANK_SNIPPET_W`, `SEARCHENG_RANK_TRUSTED_DOMAIN_BONUS`, `SEARCHENG_RANK_TLD_W`, `SEARCHENG_RANK_HTTPS_BONUS`.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        main.go                              │
│              search │ serve │ mcp                            │
└────────┬────────────┼───────┴───────────────────┬───────────┘
         │            │                           │
    ┌────▼────┐  ┌────▼────┐                 ┌────▼────┐
    │   CLI   │  │  REST   │                 │   MCP   │
    │ stdout  │  │  API    │                 │  stdio  │
    └────┬────┘  └────┬────┘                 └────┬────┘
         │            │                           │
         └────────────┼───────────────────────────┘
                      │
              ┌───────▼───────┐
              │    Engine     │
              │  Search()     │
              │  - parallel   │
              │  - merge/rank │
              │  - cache      │
              └──┬───┬───┬──┬┘
                 │   │   │  │
          ┌──────┘   │   │  └──────┐
          ▼          ▼   ▼         ▼
      ┌──────┐  ┌──────┐ ┌────┐ ┌─────┐
      │ DDG  │  │Google│ │Bing│ │Brave│
      │scrape│  │scrape│ │scrp│ │ API │
      └──────┘  └──────┘ └────┘ └─────┘
```

```
engine/
  ├── engine.go       Meta-search aggregator (parallel query, RRF + BM25F ranking)
  ├── provider.go     Provider interface
  ├── result.go       Result, Claim, TrustSignals, SearchResponse types
  ├── duckduckgo.go   DuckDuckGo HTML scraper
  ├── google.go       Google HTML scraper with CAPTCHA detection + cooldown
  ├── bing.go         Bing HTML scraper with tracking URL decoder
  ├── brave.go        Brave Search API client
  ├── answer.go       Answer extraction + NSFW filtering
  ├── claims.go       Claim extraction + cross-source corroboration
  ├── cache.go        Thread-safe in-memory cache with TTL
  ├── stopwords.go    English + Portuguese stopword lists
  └── httpclient.go   Retry, rate-limit, jitter transports + browser headers

api/
  └── server.go       REST API with CORS, RAG endpoint, graceful shutdown

mcp/
  └── server.go       MCP server (JSON-RPC 2.0 over stdio)

config/
  └── config.go       Environment-based configuration
```

## Development

```bash
# Run tests
go test ./... -race -count=1

# Run tests verbose
go test ./... -race -v

# Build
go build -o searcheng .

# Vet
go vet ./...
```

## License

[MIT](LICENSE)
