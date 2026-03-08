# SearchEng

**A meta-search engine that combines results from multiple search providers into one ranked feed.**

**Um meta-buscador que combina resultados de varios provedores de busca em um unico feed ranqueado.**

SearchEng queries DuckDuckGo, Google, Bing, and Brave in parallel, deduplicates results, ranks them using information retrieval algorithms (RRF + BM25F), and extracts featured answers and factual claims.

*SearchEng consulta DuckDuckGo, Google, Bing e Brave em paralelo, deduplica resultados, ranqueia usando algoritmos de recuperacao de informacao (RRF + BM25F) e extrai respostas e fatos verificados.*

You can use it as a **CLI tool**, a **REST API with web UI**, or an **MCP server for AI agents**.

*Voce pode usar como **ferramenta CLI**, **API REST com interface web**, ou **servidor MCP para agentes de IA**.*

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

## Table of Contents / Indice

- [Why SearchEng? / Por que SearchEng?](#why-searcheng--por-que-searcheng)
- [Getting Started / Comecando](#getting-started--comecando)
- [Usage / Uso](#usage--uso)
  - [CLI Search / Busca por CLI](#cli-search--busca-por-cli)
  - [Web UI & REST API / Interface Web e API REST](#web-ui--rest-api--interface-web-e-api-rest)
  - [MCP Server (AI Agents) / Servidor MCP (Agentes de IA)](#mcp-server-ai-agents--servidor-mcp-agentes-de-ia)
- [API Reference / Referencia da API](#api-reference--referencia-da-api)
  - [GET /search](#get-search)
  - [GET /v1/search (RAG)](#get-v1search-rag)
  - [GET /health](#get-health)
- [Configuration / Configuracao](#configuration--configuracao)
- [How It Works / Como Funciona](#how-it-works--como-funciona)
  - [Search & Ranking Pipeline](#search--ranking-pipeline)
  - [Answer Extraction / Extracao de Respostas](#answer-extraction--extracao-de-respostas)
  - [Claim Extraction / Extracao de Fatos](#claim-extraction--extracao-de-fatos)
  - [Anti-Blocking / Anti-Bloqueio](#anti-blocking--anti-bloqueio)
- [Architecture / Arquitetura](#architecture--arquitetura)
- [Development / Desenvolvimento](#development--desenvolvimento)
- [License / Licenca](#license--licenca)

---

## Why SearchEng? / Por que SearchEng?

Most search APIs are expensive, rate-limited, or require API keys. SearchEng scrapes free search engines directly and combines results into a single ranked feed.

*A maioria das APIs de busca sao caras, tem rate limit, ou exigem chaves de API. O SearchEng faz scraping direto dos buscadores gratuitos e combina os resultados em um unico feed ranqueado.*

| Use Case / Caso de Uso | How It Helps / Como Ajuda |
|---|---|
| **RAG pipelines** | `/v1/search` returns a pre-formatted `context_block` ready to inject into LLM prompts. / Retorna um `context_block` pronto para injetar em prompts de LLMs. |
| **AI agents / Agentes de IA** | MCP server lets Claude, GPT, or any MCP-compatible agent search the web. / Servidor MCP permite que Claude, GPT ou qualquer agente MCP busque na web. |
| **Research / Pesquisa** | Cross-reference results across engines with trust signals and claim corroboration. / Cruza resultados entre buscadores com sinais de confianca e corroboracao de fatos. |
| **Self-hosted search / Busca local** | No API keys required (Brave is optional). / Sem chaves de API (Brave e opcional). |

---

## Getting Started / Comecando

### Requirements / Requisitos

- **Go 1.24+** ([install Go / instalar Go](https://go.dev/dl/))
- No API keys required. Brave Search API is optional.
- *Nenhuma chave de API obrigatoria. Brave Search API e opcional.*

### Installation / Instalacao

**Option 1 / Opcao 1: `go install`**

```bash
go install github.com/KaioH3/SearchEng@latest
```

**Option 2 / Opcao 2: Build from source / Compilar do codigo-fonte**

```bash
git clone https://github.com/KaioH3/SearchEng.git
cd SearchEng
go build -o searcheng .
```

**Verify the installation / Verificar a instalacao:**

```bash
./searcheng search "hello world"
```

### Quick Start / Inicio Rapido

```bash
# Search from the terminal / Buscar pelo terminal
./searcheng search "what is WebAssembly"

# Start the web UI + API server / Iniciar a interface web + servidor API
./searcheng serve
# Open / Abrir http://localhost:8080

# Start the MCP server for AI agents / Iniciar servidor MCP para agentes de IA
./searcheng mcp
```

---

## Usage / Uso

### CLI Search / Busca por CLI

```bash
# Basic search / Busca basica
searcheng search "golang concurrency patterns"

# Choose specific providers / Escolher provedores especificos
searcheng search "rust vs go" --providers=ddg,brave

# Limit number of results / Limitar numero de resultados
searcheng search "latest news" --max-results=5

# Disable result caching / Desabilitar cache de resultados
searcheng search "breaking news" --no-cache

# Disable SafeSearch (NSFW filter) / Desabilitar SafeSearch (filtro NSFW)
searcheng search "query" --safe-search=false
```

**Available providers / Provedores disponiveis:** `ddg` (DuckDuckGo), `google`, `bing`, `brave` (requires API key / requer chave de API)

### Web UI & REST API / Interface Web e API REST

```bash
# Start with default settings (port 8080, all providers)
# Iniciar com configuracoes padrao (porta 8080, todos os provedores)
searcheng serve

# Custom port and specific providers
# Porta customizada e provedores especificos
searcheng serve --port=3000 --providers=ddg,google,bing
```

Open / Abra `http://localhost:8080` in your browser / no seu navegador.

**Web UI features / Funcionalidades da interface web:**

- Search bar with suggested queries / Barra de busca com sugestoes
- Featured answer box / Caixa de resposta destacada
- Result cards with favicons, source badges, and trust indicators / Cards de resultado com favicons, badges de fonte e indicadores de confianca
- Factual claims with confidence bars / Fatos verificados com barras de confianca
- SafeSearch toggle / Alternador de SafeSearch
- Dark/light mode (automatic, follows OS setting) / Modo escuro/claro (automatico, segue o sistema)
- Responsive design (works on mobile) / Design responsivo (funciona no celular)
- Provider status in the footer / Status dos provedores no rodape

The same server also serves the JSON API. See [API Reference](#api-reference--referencia-da-api).

*O mesmo servidor tambem serve a API JSON. Veja [Referencia da API](#api-reference--referencia-da-api).*

### MCP Server (AI Agents) / Servidor MCP (Agentes de IA)

The MCP (Model Context Protocol) server lets AI agents search the web through SearchEng.

*O servidor MCP (Model Context Protocol) permite que agentes de IA busquem na web atraves do SearchEng.*

```bash
searcheng mcp
```

**Claude Desktop** — add to / adicionar em `claude_desktop_config.json`:

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

**Claude Code** — add to / adicionar em `.mcp.json` in your project / no seu projeto:

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

**MCP `search` tool parameters / Parametros da tool `search` do MCP:**

| Parameter / Parametro | Type / Tipo | Required / Obrigatorio | Default / Padrao | Description / Descricao |
|---|---|---|---|---|
| `query` | string | Yes / Sim | — | Search query / Consulta de busca |
| `max_results` | number | No / Nao | `5` | Max results / Maximo de resultados |
| `safe_search` | boolean | No / Nao | `true` | NSFW filter / Filtro NSFW |

---

## API Reference / Referencia da API

### Endpoints

| Method / Metodo | Path / Caminho | Description / Descricao |
|---|---|---|
| `GET` | `/` | Web UI (browsers) or API info (JSON clients) / Interface web (navegadores) ou info da API (clientes JSON) |
| `GET` | `/search?q=...` | Standard search / Busca padrao |
| `GET` | `/v1/search?q=...` | RAG-optimized search / Busca otimizada para RAG |
| `GET` | `/health` | Provider health check / Status dos provedores |

The root `/` serves the web UI to browsers and JSON to API clients (based on the `Accept` header).

*A raiz `/` serve a interface web para navegadores e JSON para clientes de API (baseado no header `Accept`).*

```bash
# Get JSON from / (instead of the web UI)
# Obter JSON da / (ao inves da interface web)
curl -H "Accept: application/json" http://localhost:8080/
```

### GET /search

Standard search. Returns ranked results. / Busca padrao. Retorna resultados ranqueados.

**Parameters / Parametros:**

| Parameter | Type | Default | Description / Descricao |
|---|---|---|---|
| `q` | string | *required / obrigatorio* | Search query / Consulta |
| `page` | integer | `1` | Page number / Numero da pagina |
| `safe_search` | string | `true` | `false` or `0` to disable / para desabilitar |

**Example / Exemplo:**

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

RAG-optimized endpoint. Returns everything from `/search` plus: `answer`, `claims`, `trust_signals`, `context_block`, and `score`. This is the endpoint the web UI uses.

*Endpoint otimizado para RAG. Retorna tudo do `/search` mais: `answer`, `claims`, `trust_signals`, `context_block` e `score`. Este e o endpoint que a interface web usa.*

**Parameters / Parametros:**

| Parameter | Type | Default | Description / Descricao |
|---|---|---|---|
| `q` | string | *required / obrigatorio* | Search query / Consulta |
| `max_results` | integer | `5` | Max results (cap: 100) / Maximo de resultados (limite: 100) |
| `safe_search` | string | `true` | `false` or `0` to disable / para desabilitar |

**Example / Exemplo:**

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

**Using `context_block` in a RAG prompt / Usando `context_block` em um prompt RAG:**

```
Based on the following search results, answer the user's question.

{context_block from /v1/search}

Question: {user's question}
```

### GET /health

Returns the status of all configured providers. / Retorna o status de todos os provedores configurados.

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

## Configuration / Configuracao

All configuration is done via environment variables. No config files needed.

*Toda configuracao e feita via variaveis de ambiente. Sem arquivos de configuracao.*

### Core Settings / Configuracoes Principais

| Variable / Variavel | Default / Padrao | EN | PT-BR |
|---|---|---|---|
| `SEARCHENG_PORT` | `8080` | Server port | Porta do servidor |
| `SEARCHENG_TIMEOUT` | `5s` | Timeout per search | Timeout por busca |
| `SEARCHENG_MAX_RESULTS` | `20` | Max results returned | Maximo de resultados |
| `SEARCHENG_CACHE_TTL` | `1h` | Cache duration (`0` to disable) | Duracao do cache (`0` para desabilitar) |
| `SEARCHENG_SAFE_SEARCH` | `true` | NSFW content filter | Filtro de conteudo NSFW |

### API Keys / Chaves de API

| Variable / Variavel | Default / Padrao | EN | PT-BR |
|---|---|---|---|
| `BRAVE_API_KEY` | — | Brave Search API key ([free: 2000 queries/month](https://brave.com/search/api/)) | Chave da API Brave Search ([gratis: 2000 consultas/mes](https://brave.com/search/api/)) |

### Anti-Blocking / Anti-Bloqueio

| Variable / Variavel | Default / Padrao | EN | PT-BR |
|---|---|---|---|
| `SEARCHENG_MAX_RETRIES` | `2` | Max retries on 429/5xx | Tentativas em 429/5xx |
| `SEARCHENG_RETRY_DELAY` | `500ms` | Base delay for backoff | Delay base para backoff |
| `SEARCHENG_GOOGLE_RPM` | `1` | Google: requests/minute | Google: requisicoes/minuto |
| `SEARCHENG_DDG_RPM` | `10` | DuckDuckGo: requests/minute | DuckDuckGo: requisicoes/minuto |
| `SEARCHENG_BING_RPM` | `10` | Bing: requests/minute | Bing: requisicoes/minuto |

### Ranking Weights / Pesos de Ranqueamento

Fine-tune how results are scored. Higher values = more influence on final ranking.

*Ajuste fino de como resultados sao pontuados. Valores maiores = mais influencia no ranking final.*

| Variable / Variavel | Default / Padrao | EN | PT-BR |
|---|---|---|---|
| `SEARCHENG_RANK_POSITION_W` | `0.4` | Weight for RRF position score | Peso para score de posicao RRF |
| `SEARCHENG_RANK_BM25_W` | `0.3` | Weight for BM25F text relevance | Peso para relevancia textual BM25F |
| `SEARCHENG_RANK_MULTISOURCE_W` | `0.2` | Bonus for multi-engine results | Bonus para resultados em multiplos buscadores |
| `SEARCHENG_RANK_SNIPPET_W` | `0.1` | Weight for snippet quality | Peso para qualidade do snippet |
| `SEARCHENG_RANK_TRUSTED_DOMAIN_BONUS` | `0.5` | Bonus for trusted domains | Bonus para dominios confiaveis |
| `SEARCHENG_RANK_TLD_W` | `0.3` | Weight for TLD category | Peso para categoria do TLD |
| `SEARCHENG_RANK_HTTPS_BONUS` | `0.1` | Bonus for HTTPS | Bonus para HTTPS |

---

## How It Works / Como Funciona

### Search & Ranking Pipeline

When you make a search, here's what happens: / Quando voce faz uma busca, eis o que acontece:

```
 1. Query received / Consulta recebida
        |
 2. Check cache / Verificar cache --> hit? return cached / retornar do cache
        | (miss)
 3. Fan out to all providers in parallel (goroutines)
    Dispara para todos os provedores em paralelo (goroutines)
        |
 4. Collect results with timeout / Coletar resultados com timeout
        |
 5. Deduplicate by URL, merge sources / Deduplicar por URL, unir fontes
        |
 6. Score each result / Pontuar cada resultado:
        |   |-- RRF: Reciprocal Rank Fusion across providers
        |   |-- BM25F: text relevance (title 3x, URL 2x, snippet 1x)
        |   |-- Multi-source bonus (found in 2+ engines)
        |   |-- Trust signals (HTTPS, TLD, trusted domains)
        |   |-- Language penalty (CJK mismatch: -3.0)
        |   +-- Coverage penalty (<15% query terms: -2.0)
        |
 7. Sort by score, discard negatives / Ordenar por score, descartar negativos
        |
 8. Extract answer / Extrair resposta destacada
        |
 9. Extract claims / Extrair fatos verificaveis
        |
10. Cache results, return response / Cachear resultados, retornar resposta
```

**Scoring formula / Formula de pontuacao:**

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

### Answer Extraction / Extracao de Respostas

SearchEng picks the best sentence from the top results as a "featured answer":

*SearchEng escolhe a melhor frase dos resultados principais como "resposta destacada":*

1. Split top snippets into sentences / Divide os snippets em frases
2. Score each sentence by: query term overlap, definition patterns ("is a", "refers to"), length, source trust / Pontua cada frase por: sobreposicao com termos da consulta, padroes de definicao, tamanho, confianca da fonte
3. Highest-scoring sentence above threshold becomes the answer / A frase com maior pontuacao acima do limiar vira a resposta
4. SafeSearch filters out NSFW content / SafeSearch filtra conteudo NSFW

### Claim Extraction / Extracao de Fatos

Factual claims are statements that can be verified. SearchEng extracts them automatically:

*Fatos verificaveis sao afirmacoes que podem ser confirmadas. SearchEng os extrai automaticamente:*

1. Identify sentences with numbers, dates, comparisons, or attributions ("according to...") / Identifica frases com numeros, datas, comparacoes ou atribuicoes ("segundo...")
2. Group similar claims across sources using Jaccard similarity (threshold > 0.4) / Agrupa fatos similares entre fontes usando similaridade de Jaccard (limiar > 0.4)
3. Calculate confidence: corroboration count + trusted source bonus + claim strength / Calcula confianca: quantidade de corroboracao + bonus de fonte confiavel + forca do fato
4. Claims in multiple independent sources get higher confidence / Fatos em multiplas fontes independentes recebem confianca maior

### Anti-Blocking / Anti-Bloqueio

Scraping search engines requires careful request management. / Fazer scraping de buscadores requer gerenciamento cuidadoso de requisicoes.

| Layer / Camada | EN | PT-BR |
|---|---|---|
| **Rate limiting** | Per-provider limiter (Google: 1 req/min, DDG/Bing: 10 req/min) | Limitador por provedor (Google: 1 req/min, DDG/Bing: 10 req/min) |
| **Jittered delays** | Random 0.5-2s delay before each request | Delay aleatorio de 0.5-2s antes de cada requisicao |
| **Exponential backoff** | Auto retry on 429/5xx with increasing wait | Retry automatico em 429/5xx com espera crescente |
| **Browser mimicry** | Rotating User-Agent with matching `Sec-CH-UA` headers | User-Agent rotativo com headers `Sec-CH-UA` correspondentes |
| **Cookie jars** | Per-provider cookie persistence | Persistencia de cookies por provedor |
| **CAPTCHA detection** | Google: detects `/sorry/` redirects, 5-min cooldown | Google: detecta redirects `/sorry/`, cooldown de 5 min |
| **Consent bypass** | Google: sends consent cookies to skip EU wall | Google: envia cookies de consentimento para pular tela da UE |

Google is the most aggressive at blocking, so it uses 1 RPM and no automatic retries.

*Google e o mais agressivo em bloquear, entao usa 1 RPM e sem retries automaticos.*

---

## Architecture / Arquitetura

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

### Project Structure / Estrutura do Projeto

```
SearchEng/
|-- main.go                 Entry point / Ponto de entrada
|-- web/
|   +-- index.html          Web UI (Alpine.js + Pico CSS via go:embed)
|-- api/
|   |-- server.go           REST API (CORS, content negotiation)
|   +-- server_test.go
|-- engine/
|   |-- engine.go           Search aggregator / Agregador de busca (RRF + BM25F)
|   |-- provider.go         Provider interface / Interface de provedor
|   |-- result.go           Result, Claim, TrustSignals types / Tipos
|   |-- duckduckgo.go       DuckDuckGo scraper
|   |-- google.go           Google scraper (CAPTCHA detection + cooldown)
|   |-- bing.go             Bing scraper (tracking URL decoder)
|   |-- brave.go            Brave Search API client / Cliente API Brave
|   |-- answer.go           Answer extraction / Extracao de respostas
|   |-- claims.go           Claim extraction / Extracao de fatos
|   |-- cache.go            In-memory cache with TTL / Cache em memoria com TTL
|   |-- stopwords.go        EN + PT-BR stopword lists / Listas de stopwords
|   |-- httpclient.go       HTTP transports (retry, rate-limit, jitter)
|   +-- *_test.go           Tests / Testes
|-- mcp/
|   |-- server.go           MCP server (JSON-RPC 2.0 over stdio)
|   +-- server_test.go
+-- config/
    +-- config.go           Environment config / Configuracao por variaveis de ambiente
```

---

## Development / Desenvolvimento

```bash
# Run all tests with race detector / Rodar todos os testes com detector de race condition
go test ./... -race -count=1

# Verbose / Verboso
go test ./... -race -v

# Build / Compilar
go build -o searcheng .

# Static analysis / Analise estatica
go vet ./...

# Run in development / Rodar em desenvolvimento
go run . serve
```

**Tests / Testes:** 183 tests across 4 packages, all passing with `-race`. / 183 testes em 4 pacotes, todos passando com `-race`.

---

## License / Licenca

[MIT](LICENSE)
