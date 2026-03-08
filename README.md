# SearchEng

**A meta-search engine that combines results from multiple search providers into one ranked feed.**

SearchEng queries DuckDuckGo, Google, Bing, and Brave in parallel, deduplicates results, ranks them using information retrieval algorithms (RRF + BM25F), and extracts featured answers and factual claims.

You can use it as a **CLI tool**, a **REST API with web UI**, or an **MCP server for AI agents**.

> **Leia em Portugues:** [Clique aqui para a versao em PT-BR](#portugues)

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
- [Usage](#usage)
  - [CLI Search](#cli-search)
  - [Web UI & REST API](#web-ui--rest-api)
  - [MCP Server (AI Agents)](#mcp-server-ai-agents)
- [API Reference](#api-reference)
  - [GET /search](#get-search)
  - [GET /v1/search (RAG)](#get-v1search-rag)
  - [GET /health](#get-health)
- [Configuration](#configuration)
- [How It Works](#how-it-works)
  - [Search & Ranking Pipeline](#search--ranking-pipeline)
  - [Answer Extraction](#answer-extraction)
  - [Claim Extraction](#claim-extraction)
  - [Anti-Blocking](#anti-blocking)
- [Architecture](#architecture)
- [Development](#development)
- [License](#license)
- [Portugues](#portugues)

---

## Why SearchEng?

Most search APIs are expensive, rate-limited, or require API keys. SearchEng scrapes free search engines directly and combines results into a single ranked feed.

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

The root `/` serves the web UI to browsers and returns JSON to API clients (based on the `Accept` header). To get JSON:

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

**Ranking weights:**

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
        |
 2. Check cache --> hit? return cached results
        | (miss)
 3. Fan out to all providers in parallel (goroutines)
        |
 4. Collect results (with timeout)
        |
 5. Deduplicate by URL (merge sources)
        |
 6. Score each result:
        |   |-- RRF: Reciprocal Rank Fusion across providers
        |   |-- BM25F: text relevance (title 3x, URL 2x, snippet 1x)
        |   |-- Multi-source bonus (found in 2+ engines)
        |   |-- Trust signals (HTTPS, TLD, trusted domains)
        |   |-- Language penalty (CJK mismatch: -3.0)
        |   +-- Coverage penalty (<15% query terms: -2.0)
        |
 7. Sort by score, discard negatives
        |
 8. Extract answer (best snippet sentence)
        |
 9. Extract claims (cross-source corroboration)
        |
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
                      +---------------------+
                      |      main.go        |
                      |  search | serve| mcp|
                      +---+-----+------+----+
                          |     |      |
                     +----+     |      +----+
                     v          v           v
                +---------+ +--------+ +--------+
                |   CLI   | |  REST  | |  MCP   |
                | stdout  | | API +  | | stdio  |
                |         | | Web UI | |        |
                +----+----+ +---+----+ +---+----+
                     |          |          |
                     +----------+----------+
                                |
                        +-------v-------+
                        |    Engine     |
                        |  - parallel   |
                        |  - rank/merge |
                        |  - answer     |
                        |  - claims     |
                        |  - cache      |
                        +--+--+--+--+---+
                           |  |  |  |
                    +------+  |  |  +------+
                    v         v  v         v
                +------+ +------+ +----+ +-----+
                | DDG  | |Google| |Bing| |Brave|
                |scrape| |scrape| |scrp| | API |
                +------+ +------+ +----+ +-----+
```

**Project structure:**

```
SearchEng/
|-- main.go                 Entry point: CLI, serve, and MCP commands
|-- web/
|   +-- index.html          Web UI (Alpine.js + Pico CSS, served via go:embed)
|-- api/
|   |-- server.go           REST API server with CORS and content negotiation
|   +-- server_test.go      API tests
|-- engine/
|   |-- engine.go           Search aggregator (parallel query, RRF + BM25F ranking)
|   |-- provider.go         Provider interface
|   |-- result.go           Result, Claim, TrustSignals types
|   |-- duckduckgo.go       DuckDuckGo HTML scraper
|   |-- google.go           Google HTML scraper (CAPTCHA detection + cooldown)
|   |-- bing.go             Bing HTML scraper (tracking URL decoder)
|   |-- brave.go            Brave Search API client
|   |-- answer.go           Featured answer extraction
|   |-- claims.go           Factual claim extraction + corroboration
|   |-- cache.go            Thread-safe in-memory cache with TTL
|   |-- stopwords.go        English + Portuguese stopword lists
|   |-- httpclient.go       HTTP transports (retry, rate-limit, jitter, browser headers)
|   +-- *_test.go           Tests for each component
|-- mcp/
|   |-- server.go           MCP server (JSON-RPC 2.0 over stdio)
|   +-- server_test.go      MCP tests
+-- config/
    +-- config.go           Environment-based configuration
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

---

---

<a id="portugues"></a>

# SearchEng (Portugues)

**Um meta-buscador que combina resultados de varios provedores de busca em um unico feed ranqueado.**

O SearchEng consulta DuckDuckGo, Google, Bing e Brave em paralelo, deduplica resultados, ranqueia usando algoritmos de recuperacao de informacao (RRF + BM25F) e extrai respostas destacadas e fatos verificaveis.

Voce pode usar como **ferramenta CLI**, **API REST com interface web**, ou **servidor MCP para agentes de IA**.

> **Read in English:** [Click here for the English version](#searcheng)

```
$ searcheng search "como funciona garbage collection em Go"

Searching for: como funciona garbage collection em Go
Providers: 3 active
────────────────────────────────────────────────────────────
  DuckDuckGo: 12 results
  Bing: 10 results
  Google: 8 results
────────────────────────────────────────────────────────────

💡 Go usa um garbage collector concorrente tri-color mark-and-sweep que roda
   ao lado das goroutines com pause times abaixo de 1ms.
────────────────────────────────────────────────────────────

1. A Guide to the Go Garbage Collector
   https://go.dev/doc/gc-guide
   Go's garbage collector is a non-generational concurrent collector...
   Score: 4.82 | Sources: DuckDuckGo, Google, Bing

Found 18 results in 1.2s
```

---

## Indice

- [Por que SearchEng?](#por-que-searcheng)
- [Comecando](#comecando)
- [Uso](#uso)
  - [Busca por CLI](#busca-por-cli)
  - [Interface Web e API REST](#interface-web-e-api-rest)
  - [Servidor MCP (Agentes de IA)](#servidor-mcp-agentes-de-ia)
- [Referencia da API](#referencia-da-api)
  - [GET /search](#get-search-1)
  - [GET /v1/search (RAG)](#get-v1search-rag-1)
  - [GET /health](#get-health-1)
- [Configuracao](#configuracao)
- [Como Funciona](#como-funciona)
  - [Pipeline de Busca e Ranqueamento](#pipeline-de-busca-e-ranqueamento)
  - [Extracao de Respostas](#extracao-de-respostas)
  - [Extracao de Fatos](#extracao-de-fatos)
  - [Anti-Bloqueio](#anti-bloqueio)
- [Arquitetura](#arquitetura)
- [Desenvolvimento](#desenvolvimento)
- [Licenca](#licenca)

---

## Por que SearchEng?

A maioria das APIs de busca sao caras, tem rate limit, ou exigem chaves de API. O SearchEng faz scraping direto dos buscadores gratuitos e combina os resultados em um unico feed ranqueado.

| Caso de Uso | Como o SearchEng Ajuda |
|---|---|
| **Pipelines RAG** | `/v1/search` retorna um `context_block` pre-formatado, pronto para injetar em prompts de LLMs |
| **Agentes de IA** | Servidor MCP permite que Claude, GPT ou qualquer agente compativel busque na web |
| **Pesquisa** | Cruza resultados entre buscadores com sinais de confianca e corroboracao de fatos |
| **Busca local** | Sem chaves de API obrigatorias (Brave e opcional para mais qualidade) |

---

## Comecando

### Requisitos

- **Go 1.24+** ([instalar Go](https://go.dev/dl/))
- Nenhuma chave de API obrigatoria (Brave Search API e opcional)

### Instalacao

**Opcao 1: `go install`**

```bash
go install github.com/KaioH3/SearchEng@latest
```

**Opcao 2: Compilar do codigo-fonte**

```bash
git clone https://github.com/KaioH3/SearchEng.git
cd SearchEng
go build -o searcheng .
```

Verificar a instalacao:

```bash
./searcheng search "hello world"
```

### Inicio Rapido

```bash
# Buscar pelo terminal
./searcheng search "what is WebAssembly"

# Iniciar a interface web + servidor API
./searcheng serve
# Abrir http://localhost:8080 no navegador

# Iniciar servidor MCP para agentes de IA
./searcheng mcp
```

---

## Uso

### Busca por CLI

```bash
# Busca basica
searcheng search "golang concurrency patterns"

# Escolher provedores especificos
searcheng search "rust vs go" --providers=ddg,brave

# Limitar numero de resultados
searcheng search "latest news" --max-results=5

# Desabilitar cache de resultados
searcheng search "breaking news" --no-cache

# Desabilitar SafeSearch (filtro NSFW)
searcheng search "query" --safe-search=false
```

**Provedores disponiveis:** `ddg` (DuckDuckGo), `google`, `bing`, `brave` (requer chave de API)

### Interface Web e API REST

```bash
# Iniciar com configuracoes padrao (porta 8080, todos os provedores)
searcheng serve

# Porta customizada e provedores especificos
searcheng serve --port=3000 --providers=ddg,google,bing
```

Abra `http://localhost:8080` no seu navegador. A interface inclui:

- Barra de busca com sugestoes de consulta
- Caixa de resposta destacada
- Cards de resultado com favicons, badges de fonte e indicadores de confianca
- Fatos verificados com barras de confianca
- Alternador de SafeSearch
- Modo escuro/claro (automatico, segue o sistema operacional)
- Design responsivo (funciona no celular)
- Status dos provedores no rodape

O mesmo servidor tambem serve a API JSON. Veja [Referencia da API](#referencia-da-api) para detalhes.

### Servidor MCP (Agentes de IA)

O servidor MCP (Model Context Protocol) permite que agentes de IA busquem na web atraves do SearchEng.

```bash
searcheng mcp
```

**Claude Desktop** — adicionar em `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "searcheng": {
      "command": "/caminho/para/searcheng",
      "args": ["mcp"]
    }
  }
}
```

**Claude Code** — adicionar em `.mcp.json` no seu projeto:

```json
{
  "mcpServers": {
    "searcheng": {
      "command": "/caminho/para/searcheng",
      "args": ["mcp"]
    }
  }
}
```

Parametros da tool `search` do MCP:

| Parametro | Tipo | Obrigatorio | Padrao | Descricao |
|---|---|---|---|---|
| `query` | string | Sim | — | Consulta de busca |
| `max_results` | number | Nao | `5` | Maximo de resultados |
| `safe_search` | boolean | Nao | `true` | Filtro NSFW |

---

## Referencia da API

### Endpoints

| Metodo | Caminho | Descricao |
|---|---|---|
| `GET` | `/` | Interface web (navegadores) ou info da API (clientes JSON) |
| `GET` | `/search?q=...` | Busca padrao com resultados ranqueados |
| `GET` | `/v1/search?q=...` | Busca otimizada para RAG com answer, claims e trust signals |
| `GET` | `/health` | Status dos provedores |

A raiz `/` serve a interface web para navegadores e JSON para clientes de API (baseado no header `Accept`). Para obter JSON:

```bash
curl -H "Accept: application/json" http://localhost:8080/
```

### GET /search

Endpoint de busca padrao. Retorna resultados ranqueados.

**Parametros:**

| Parametro | Tipo | Padrao | Descricao |
|---|---|---|---|
| `q` | string | *obrigatorio* | Consulta de busca |
| `page` | integer | `1` | Numero da pagina |
| `safe_search` | string | `true` | `false` ou `0` para desabilitar |

**Exemplo:**

```bash
curl "http://localhost:8080/search?q=golang+error+handling&page=1"
```

A resposta segue o mesmo formato JSON da [versao em ingles](#get-search).

### GET /v1/search (RAG)

Endpoint otimizado para RAG. Retorna tudo do `/search` mais: `answer`, `claims`, `trust_signals`, `context_block` e `score`.

Este e o endpoint que a interface web usa.

**Parametros:**

| Parametro | Tipo | Padrao | Descricao |
|---|---|---|---|
| `q` | string | *obrigatorio* | Consulta de busca |
| `max_results` | integer | `5` | Maximo de resultados (limite: 100) |
| `safe_search` | string | `true` | `false` ou `0` para desabilitar |

**Exemplo:**

```bash
curl "http://localhost:8080/v1/search?q=golang+error+handling&max_results=3"
```

A resposta segue o mesmo formato JSON da [versao em ingles](#get-v1search-rag).

**Usando `context_block` em um prompt RAG:**

```
Com base nos seguintes resultados de busca, responda a pergunta do usuario.

{context_block do /v1/search}

Pergunta: {pergunta do usuario}
```

### GET /health

Retorna o status de todos os provedores configurados.

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

## Configuracao

Toda configuracao e feita via variaveis de ambiente. Sem arquivos de configuracao.

**Configuracoes principais:**

| Variavel | Padrao | Descricao |
|---|---|---|
| `SEARCHENG_PORT` | `8080` | Porta do servidor |
| `SEARCHENG_TIMEOUT` | `5s` | Timeout por busca |
| `SEARCHENG_MAX_RESULTS` | `20` | Maximo de resultados |
| `SEARCHENG_CACHE_TTL` | `1h` | Duracao do cache (`0` para desabilitar) |
| `SEARCHENG_SAFE_SEARCH` | `true` | Filtro de conteudo NSFW |

**Chaves de API:**

| Variavel | Padrao | Descricao |
|---|---|---|
| `BRAVE_API_KEY` | — | Chave da API Brave Search ([gratis: 2000 consultas/mes](https://brave.com/search/api/)) |

**Anti-bloqueio:**

| Variavel | Padrao | Descricao |
|---|---|---|
| `SEARCHENG_MAX_RETRIES` | `2` | Tentativas em erros 429/5xx |
| `SEARCHENG_RETRY_DELAY` | `500ms` | Delay base para backoff exponencial |
| `SEARCHENG_GOOGLE_RPM` | `1` | Google: requisicoes por minuto |
| `SEARCHENG_DDG_RPM` | `10` | DuckDuckGo: requisicoes por minuto |
| `SEARCHENG_BING_RPM` | `10` | Bing: requisicoes por minuto |

**Pesos de ranqueamento:**

Ajuste fino de como resultados sao pontuados. Valores maiores = mais influencia no ranking final.

| Variavel | Padrao | Descricao |
|---|---|---|
| `SEARCHENG_RANK_POSITION_W` | `0.4` | Peso para score de posicao RRF |
| `SEARCHENG_RANK_BM25_W` | `0.3` | Peso para relevancia textual BM25F |
| `SEARCHENG_RANK_MULTISOURCE_W` | `0.2` | Bonus para resultados em multiplos buscadores |
| `SEARCHENG_RANK_SNIPPET_W` | `0.1` | Peso para qualidade do snippet |
| `SEARCHENG_RANK_TRUSTED_DOMAIN_BONUS` | `0.5` | Bonus para dominios confiaveis (GitHub, Wikipedia, etc.) |
| `SEARCHENG_RANK_TLD_W` | `0.3` | Peso para categoria do TLD |
| `SEARCHENG_RANK_HTTPS_BONUS` | `0.1` | Bonus para HTTPS |

---

## Como Funciona

### Pipeline de Busca e Ranqueamento

Quando voce faz uma busca, eis o que acontece:

```
 1. Consulta recebida
        |
 2. Verificar cache --> encontrou? retornar do cache
        | (miss)
 3. Disparar para todos os provedores em paralelo (goroutines)
        |
 4. Coletar resultados (com timeout)
        |
 5. Deduplicar por URL (unir fontes)
        |
 6. Pontuar cada resultado:
        |   |-- RRF: Reciprocal Rank Fusion entre provedores
        |   |-- BM25F: relevancia textual (titulo 3x, URL 2x, snippet 1x)
        |   |-- Bonus multi-fonte (encontrado em 2+ buscadores)
        |   |-- Sinais de confianca (HTTPS, TLD, dominios confiaveis)
        |   |-- Penalidade de idioma (incompatibilidade CJK: -3.0)
        |   +-- Penalidade de cobertura (<15% dos termos: -2.0)
        |
 7. Ordenar por score, descartar negativos
        |
 8. Extrair resposta (melhor frase dos snippets)
        |
 9. Extrair fatos verificaveis (corroboracao cruzada)
        |
10. Cachear resultados, retornar resposta
```

**Formula de pontuacao:**

```
score = PositionW    * RRF * 100
      + BM25W        * BM25F(titulo*3, snippet*1, url*2)
      + MultiSourceW * (quantidadeFontes - 1) * 10
      + SnippetW     * qualidadeSnippet
      + bonusDominioConfiavel
      + scoreTLD           (.edu/.gov: +1.0, .org: +0.5, .xyz/.click: -0.5)
      + bonusHTTPS
      + penalidadeIdioma   (incompatibilidade CJK: -3.0)
      + penalidadeCobertura (baixa cobertura da consulta: -2.0)
```

### Extracao de Respostas

O SearchEng escolhe a melhor frase dos resultados principais como "resposta destacada":

1. Divide os snippets dos melhores resultados em frases
2. Pontua cada frase por: sobreposicao com termos da consulta, padroes de definicao ("is a", "refers to"), tamanho e confianca da fonte
3. A frase com maior pontuacao acima do limiar vira a resposta destacada
4. SafeSearch filtra conteudo NSFW das respostas

### Extracao de Fatos

Fatos verificaveis sao afirmacoes que podem ser confirmadas. O SearchEng os extrai automaticamente:

1. Identifica frases contendo numeros, datas, comparacoes ou atribuicoes ("segundo...", "de acordo com...")
2. Agrupa fatos similares entre fontes usando similaridade de Jaccard (limiar > 0.4)
3. Calcula confianca baseado em: quantidade de corroboracao, bonus de fonte confiavel e forca do fato
4. Fatos encontrados em multiplas fontes independentes recebem confianca maior

### Anti-Bloqueio

Fazer scraping de buscadores requer gerenciamento cuidadoso de requisicoes. O SearchEng usa multiplas camadas:

| Camada | O que Faz |
|---|---|
| **Rate limiting** | Limitador por provedor (Google: 1 req/min, DDG/Bing: 10 req/min) |
| **Delays com jitter** | Delay aleatorio de 0.5-2s antes de cada requisicao para parecer natural |
| **Backoff exponencial** | Retry automatico em 429/5xx com tempos de espera crescentes |
| **Mimicry de navegador** | User-Agent rotativo com headers `Sec-CH-UA` e `Sec-Fetch-*` correspondentes |
| **Cookie jars** | Persistencia de cookies por provedor (mantem contexto de sessao) |
| **Deteccao de CAPTCHA** | Google: detecta redirects `/sorry/` e paginas de CAPTCHA, ativa cooldown de 5 min |
| **Bypass de consentimento** | Google: envia cookies de consentimento para pular tela de consentimento da UE |

O Google e o mais agressivo em bloquear, entao usa rate limit mais restrito (1 RPM) e sem retries automaticos.

---

## Arquitetura

```
                      +---------------------+
                      |      main.go        |
                      |  search | serve| mcp|
                      +---+-----+------+----+
                          |     |      |
                     +----+     |      +----+
                     v          v           v
                +---------+ +--------+ +--------+
                |   CLI   | |  REST  | |  MCP   |
                | stdout  | | API +  | | stdio  |
                |         | | Web UI | |        |
                +----+----+ +---+----+ +---+----+
                     |          |          |
                     +----------+----------+
                                |
                        +-------v-------+
                        |    Engine     |
                        |  - paralelo  |
                        |  - rank/merge|
                        |  - answer    |
                        |  - claims    |
                        |  - cache     |
                        +--+--+--+--+--+
                           |  |  |  |
                    +------+  |  |  +------+
                    v         v  v         v
                +------+ +------+ +----+ +-----+
                | DDG  | |Google| |Bing| |Brave|
                |scrape| |scrape| |scrp| | API |
                +------+ +------+ +----+ +-----+
```

**Estrutura do projeto:**

```
SearchEng/
|-- main.go                 Ponto de entrada: comandos CLI, serve e MCP
|-- web/
|   +-- index.html          Interface web (Alpine.js + Pico CSS via go:embed)
|-- api/
|   |-- server.go           Servidor REST API com CORS e content negotiation
|   +-- server_test.go      Testes da API
|-- engine/
|   |-- engine.go           Agregador de busca (consulta paralela, ranking RRF + BM25F)
|   |-- provider.go         Interface de provedor
|   |-- result.go           Tipos: Result, Claim, TrustSignals
|   |-- duckduckgo.go       Scraper do DuckDuckGo
|   |-- google.go           Scraper do Google (deteccao de CAPTCHA + cooldown)
|   |-- bing.go             Scraper do Bing (decodificador de URL de tracking)
|   |-- brave.go            Cliente da API Brave Search
|   |-- answer.go           Extracao de resposta destacada
|   |-- claims.go           Extracao de fatos + corroboracao
|   |-- cache.go            Cache em memoria thread-safe com TTL
|   |-- stopwords.go        Listas de stopwords EN + PT-BR
|   |-- httpclient.go       Transports HTTP (retry, rate-limit, jitter, headers de navegador)
|   +-- *_test.go           Testes de cada componente
|-- mcp/
|   |-- server.go           Servidor MCP (JSON-RPC 2.0 via stdio)
|   +-- server_test.go      Testes do MCP
+-- config/
    +-- config.go           Configuracao baseada em variaveis de ambiente
```

---

## Desenvolvimento

```bash
# Rodar todos os testes (com detector de race condition)
go test ./... -race -count=1

# Testes com saida verbosa
go test ./... -race -v

# Compilar o binario
go build -o searcheng .

# Analise estatica
go vet ./...

# Rodar o servidor em desenvolvimento
go run . serve
```

**Testes:** 183 testes em 4 pacotes, todos passando com `-race`.

---

## Licenca

[MIT](LICENSE)
