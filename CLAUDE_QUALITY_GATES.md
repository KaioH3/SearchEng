# CLAUDE_QUALITY_GATES.md
# Quality Gates — Pre-PR Checklist + Review Standards

> Run every item before opening a PR. If any item fails, fix it first.
> Language-agnostic. Extracted from real human review patterns.

---

## GATE 1 — Internationalization (i18n)
*Priority #1 — most commonly missed, most visible to end users*

- [ ] **Every** user-facing string uses the i18n function (`t('key')`, `_()`, `gettext()`, etc.)
- [ ] **All supported locales** have the new keys (never leave a locale behind)
- [ ] **No orphan keys** — keys removed from code are also removed from locale files
- [ ] **No missing keys** — keys added to code are added to ALL locale files
- [ ] Interpolated variables work in all locales (`{{name}}`, `%s`, `{0}`)
- [ ] New namespaces/domains registered in the i18n configuration file
- [ ] Backend **never** stores user-facing text — stores codes, resolves to text at render

**The rule:** If a user can read it, it must be translated. No exceptions.

---

## GATE 2 — Backend Correctness

- [ ] Every API function/endpoint has input validation (schema, typed, not `any`)
- [ ] Validation uses a library (Zod, Pydantic, Joi, class-validator) — never manual `if` chains
- [ ] Errors use a typed enum or constant — never raw strings like `"user_not_found"`
- [ ] Database queries are bounded — `.take(N)` / `.limit(N)` / `.first()` — never unbounded `.findAll()`
- [ ] Mutations return a typed result (success + data, or error + message)
- [ ] Queries propagate errors through the framework's error state — not a Result wrapper
- [ ] No secrets or credentials hardcoded — environment variables only
- [ ] No retry logic on 4xx errors — only on transient 5xx / network failures
- [ ] Auth/permission check on every protected mutation and query

**OOM risk:** Unbounded queries on large collections will crash production. Always paginate.

---

## GATE 3 — Frontend Correctness

- [ ] Mutations/writes go through project wrappers, not raw library hooks
- [ ] Queries use project wrappers, not raw library imports
- [ ] Frontend does NOT import from server internals (only generated types / shared packages)
- [ ] Shared types are in a shared package — not duplicated between frontend and backend
- [ ] Typed routes / navigation paths — no string literals for routes
- [ ] HTTP callbacks and webhooks have authentication middleware

**Why wrappers matter:** Project wrappers contain cross-cutting logic (error handling, auth, loading states, cache). Bypassing them introduces inconsistency silently.

---

## GATE 4 — UI/UX Quality

- [ ] Styling via design system / utility classes — never inline `style={{}}` or `style=""`
- [ ] Primary colors, spacing, typography from design tokens — never hardcoded hex values
- [ ] Long lists wrapped in a scrollable container (mobile doesn't scroll automatically)
- [ ] Existing UI components reused — not recreated from scratch
- [ ] Loading state shown for every async operation (button disabled, spinner, skeleton)
- [ ] Error state shown and actionable (not silent failure)
- [ ] Empty state shown when list/content is empty
- [ ] Responsive — tested on mobile AND desktop (if applicable)
- [ ] Touch targets large enough on mobile (minimum 44×44px)
- [ ] Forms have proper labels, accessible and keyboard-navigable

**Design system first:** Before building any UI, check what components already exist.

---

## GATE 5 — Code Quality

- [ ] Zero dead code: no unused imports, unused variables, unused functions
- [ ] No `any` / untyped values — explicit types or correct inference
- [ ] No `console.log` / `print` / debug statements in production code
- [ ] No commented-out code blocks left in the diff
- [ ] Existing functions reused — searched before creating new ones
- [ ] Stable identifiers (IDs, keys, codes) are NEVER overwritten with display values
- [ ] Display values live in a separate field (`label`, `displayName`, `title`)
- [ ] No duplicate imports — merged into one import statement per module

---

## GATE 6 — Testing (TDD / BDD / DDD / E2E / Smoke)

### 6.1 — Static Analysis (runs first, blocks everything)
- [ ] `type-check` passes — zero type errors
- [ ] `lint` passes — zero new lint errors in changed files
- [ ] `format:check` passes — code is formatted
- [ ] `vet` / `staticcheck` passes (Go), `mypy --strict` (Python), `tsc --noEmit` (TS)

### 6.2 — Unit Tests (TDD — Test-Driven Development)
*Write the test BEFORE the implementation. Red → Green → Refactor.*

- [ ] Every new public function has at least one happy path and one failure path test
- [ ] Tests assert **behavior**, not implementation details
- [ ] Tests are **deterministic** — no flaky tests, no time-dependent assertions
- [ ] Test fixtures / factories used — no hardcoded IDs or magic values
- [ ] External dependencies are **mocked/stubbed** — unit tests never hit network, disk, or DB
- [ ] Test names describe the scenario: `Test_NormalizeURL_StripsTrailingSlash`, not `TestNormalize1`
- [ ] Edge cases covered: empty input, nil/null, boundary values, unicode, max-length strings
- [ ] Error paths tested: what happens when the dependency fails?

**TDD cycle:**
```
1. RED    — Write a failing test that describes the desired behavior
2. GREEN  — Write the minimum code to make the test pass
3. REFACTOR — Clean up without changing behavior, tests stay green
```

**TDD rules:**
- Never write production code without a failing test first
- Never write more test code than needed to fail (compilation counts as failure)
- Never write more production code than needed to pass the current test
- Each cycle should take 1-5 minutes — if longer, the step is too big

```go
// Example (Go): TDD for URL normalization
func TestNormalizeURL_StripsFragment(t *testing.T) {
    got := normalizeURL("https://example.com/page#section")
    want := "https://example.com/page"
    if got != want {
        t.Errorf("normalizeURL() = %q, want %q", got, want)
    }
}

func TestNormalizeURL_LowercasesHost(t *testing.T) {
    got := normalizeURL("https://EXAMPLE.COM/Path")
    want := "https://example.com/Path"
    if got != want {
        t.Errorf("normalizeURL() = %q, want %q", got, want)
    }
}
```

```python
# Example (Python): TDD with pytest
def test_search_returns_empty_for_unknown_query():
    engine = SearchEngine(providers=[MockProvider(results=[])])
    response = engine.search("xyznonexistent", page=1)
    assert response.results == []
    assert response.query == "xyznonexistent"

def test_search_deduplicates_by_url():
    p1 = MockProvider(results=[Result(url="https://a.com", title="A")])
    p2 = MockProvider(results=[Result(url="https://a.com", title="A duplicate")])
    engine = SearchEngine(providers=[p1, p2])
    response = engine.search("test", page=1)
    assert len(response.results) == 1
```

```typescript
// Example (TypeScript): TDD with vitest
describe('mergeAndRank', () => {
  it('boosts results appearing in multiple providers', () => {
    const results = mergeAndRank([
      { provider: 'ddg', results: [{ url: 'https://a.com', title: 'A' }] },
      { provider: 'bing', results: [{ url: 'https://a.com', title: 'A' }] },
    ]);
    expect(results[0].score).toBeGreaterThan(10);
    expect(results[0].source).toContain('ddg');
  });
});
```

### 6.3 — BDD (Behavior-Driven Development)
*Describe behavior in natural language FIRST, then automate.*

- [ ] Features described in **Given/When/Then** format before implementation
- [ ] Scenarios cover the user's perspective, not the developer's
- [ ] Scenarios are **executable** — mapped to automated tests
- [ ] Each scenario tests ONE behavior — not a combined workflow
- [ ] Non-technical stakeholders can read and validate the scenarios

**BDD workflow:**
```
1. DISCOVER  — Discuss the feature with stakeholders, extract examples
2. FORMULATE — Write scenarios in Given/When/Then (Gherkin or plain text)
3. AUTOMATE  — Implement step definitions that execute the scenarios
4. VERIFY    — Run scenarios as part of CI
```

**Gherkin format (Cucumber, Godog, Behave, etc.):**
```gherkin
Feature: Meta-search aggregation
  As a user searching the web
  I want results from multiple search engines
  So that I get comprehensive and unbiased results

  Scenario: Successful multi-provider search
    Given the search engine has providers "DuckDuckGo" and "Bing"
    And both providers are online and responding
    When I search for "golang tutorial"
    Then I should receive results from both providers
    And duplicate URLs should be merged into a single result
    And merged results should have a higher ranking score

  Scenario: Graceful degradation when a provider fails
    Given the search engine has providers "DuckDuckGo" and "Google"
    And "Google" is returning HTTP 503
    When I search for "test query"
    Then I should still receive results from "DuckDuckGo"
    And I should see a warning about "Google" failure

  Scenario: Search timeout returns partial results
    Given the search engine has a timeout of 2 seconds
    And "Bing" takes 5 seconds to respond
    When I search for "slow query"
    Then I should receive results before 2 seconds
    And results from fast providers should be included

  Scenario: Empty query is rejected
    Given the search engine is running
    When I search for ""
    Then I should receive an error "missing query"
    And no provider should be contacted
```

**Step definition example (Go with godog):**
```go
func (s *SearchSuite) theSearchEngineHasProviders(p1, p2 string) error {
    s.engine = &Engine{
        Providers: []Provider{s.mockProvider(p1), s.mockProvider(p2)},
        Timeout:   5 * time.Second,
    }
    return nil
}

func (s *SearchSuite) iSearchFor(query string) error {
    s.response = s.engine.Search(query, 1)
    return nil
}

func (s *SearchSuite) iShouldReceiveResultsFromBothProviders() error {
    if len(s.response.Results) == 0 {
        return fmt.Errorf("expected results, got none")
    }
    return nil
}
```

**BDD anti-patterns to avoid:**
- Writing scenarios AFTER the code (that's just documentation, not BDD)
- Scenarios with technical jargon ("When the goroutine sends to channel...")
- Scenarios testing implementation ("Then the map should have 3 entries")
- One mega-scenario testing 10 things — split into focused scenarios

### 6.4 — Integration Tests
*Test real interactions between components — no mocks at this layer.*

- [ ] Integration tests use **real** dependencies (test DB, test HTTP server, temp files)
- [ ] Each test sets up and tears down its own state — no shared mutable state between tests
- [ ] Database tests use transactions with rollback, or a test-specific database
- [ ] HTTP tests use `httptest.NewServer` (Go), `supertest` (Node), `TestClient` (FastAPI)
- [ ] Tests verify the **contract** between layers, not internal logic
- [ ] Slow tests are tagged and can be skipped in fast feedback loops: `go test -short`

```go
// Example (Go): Integration test for the REST API
func TestSearchEndpoint_ReturnsJSON(t *testing.T) {
    eng := &engine.Engine{
        Providers:  []engine.Provider{&mockProvider{results: sampleResults}},
        Timeout:    5 * time.Second,
        MaxResults: 10,
    }
    srv := &api.Server{Engine: eng, Port: 0}
    ts := httptest.NewServer(http.HandlerFunc(srv.HandleSearch))
    defer ts.Close()

    resp, err := http.Get(ts.URL + "/search?q=test")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("status = %d, want 200", resp.StatusCode)
    }
    if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
        t.Errorf("Content-Type = %q, want application/json", ct)
    }
}
```

### 6.5 — DDD (Domain-Driven Design) Testing Strategy
*Tests follow domain boundaries. Each Bounded Context has its own test suite.*

- [ ] Tests organized by **domain/bounded context**, not by technical layer
- [ ] Domain entities have **invariant tests** — verify business rules can never be violated
- [ ] Value Objects are tested for **equality and immutability**
- [ ] Aggregates are tested through their **public root methods** only
- [ ] Domain Events are tested: when X happens, event Y is emitted
- [ ] Application Services (use cases) are tested with mocked repositories
- [ ] Infrastructure (repos, HTTP clients) tested in integration layer

**DDD test organization:**
```
tests/
├── domain/                    # Pure domain logic — no I/O, no framework
│   ├── search/
│   │   ├── result_test.go     # Value Object: Result equality, immutability
│   │   ├── engine_test.go     # Aggregate: merge, rank, deduplicate
│   │   └── provider_test.go   # Interface contract tests
│   └── ranking/
│       └── scorer_test.go     # Domain Service: scoring algorithm
├── application/               # Use cases — mocked repos/providers
│   ├── search_usecase_test.go # Orchestration: search → aggregate → return
│   └── cache_usecase_test.go  # Caching behavior
├── infrastructure/            # Real I/O — integration tests
│   ├── duckduckgo_test.go     # HTTP scraping (use recorded responses)
│   ├── bing_test.go
│   ├── google_test.go
│   └── api_server_test.go     # REST API endpoints
├── e2e/                       # Full system tests
│   └── search_flow_test.go    # CLI or HTTP end-to-end
└── fixtures/                  # Shared test data
    ├── testdata/
    │   ├── duckduckgo_response.html
    │   ├── bing_response.html
    │   └── google_response.html
    └── factories.go           # Test object builders
```

**DDD testing principles:**
- Domain layer tests are **fast and pure** — they never touch I/O
- If you need I/O in a domain test, your domain is leaking infrastructure concerns
- Test the **aggregate root**, not internal entities directly
- Use the **Ports & Adapters** pattern: domain defines interfaces, infrastructure implements them
- Test infrastructure adapters against **recorded/fixture data** (golden files)

```go
// Example: Domain layer test — pure logic, no I/O
func TestMergeAndRank_BoostsDuplicates(t *testing.T) {
    eng := &Engine{MaxResults: 20}
    results := eng.mergeAndRank([]providerResult{
        {provider: "ddg", results: []Result{{URL: "https://a.com", Title: "A", Source: "DDG"}}},
        {provider: "bing", results: []Result{{URL: "https://a.com", Title: "A", Source: "Bing"}}},
    })
    if len(results) != 1 {
        t.Fatalf("expected 1 merged result, got %d", len(results))
    }
    if results[0].Score <= 10 {
        t.Errorf("expected boosted score > 10, got %f", results[0].Score)
    }
}

// Example: Infrastructure test — uses recorded HTML fixture
func TestDuckDuckGo_ParseFixture(t *testing.T) {
    f, _ := os.Open("testdata/duckduckgo_response.html")
    defer f.Close()
    ddg := &DuckDuckGo{}
    results, err := ddg.parse(f)
    if err != nil {
        t.Fatal(err)
    }
    if len(results) == 0 {
        t.Error("expected results from fixture, got none")
    }
    if results[0].URL == "" {
        t.Error("first result has empty URL")
    }
}
```

### 6.6 — Smoke Tests
*Minimal sanity check: "does it turn on and not explode?"*

- [ ] Smoke tests run FIRST in CI — fail fast before expensive test suites
- [ ] Cover critical paths only: app starts, health check responds, basic search returns data
- [ ] Should complete in under 30 seconds
- [ ] No complex assertions — just "it works" or "it doesn't"
- [ ] Run after every deployment to production

```bash
# Smoke test script example
#!/bin/bash
set -e

echo "=== SMOKE TEST ==="

# 1. Binary exists and runs
./searcheng --help > /dev/null 2>&1 || { echo "FAIL: binary won't run"; exit 1; }
echo "PASS: binary runs"

# 2. Health check (if server mode)
./searcheng serve --port=9999 &
PID=$!
sleep 2
STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9999/health)
kill $PID 2>/dev/null
[ "$STATUS" = "200" ] || { echo "FAIL: health check returned $STATUS"; exit 1; }
echo "PASS: health endpoint returns 200"

# 3. Basic search returns results
RESULTS=$(./searcheng search "test" --providers=ddg --max-results=3 2>/dev/null | grep -c "http")
[ "$RESULTS" -gt 0 ] || { echo "FAIL: search returned no results"; exit 1; }
echo "PASS: search returns results"

echo "=== ALL SMOKE TESTS PASSED ==="
```

```go
// Smoke test in Go
func TestSmoke_EngineDoesNotPanic(t *testing.T) {
    eng := &Engine{
        Providers:  []Provider{&DuckDuckGo{Client: &http.Client{Timeout: 10 * time.Second}}},
        Timeout:    10 * time.Second,
        MaxResults: 5,
    }
    resp := eng.Search("test", 1)
    if resp.Query != "test" {
        t.Errorf("query = %q, want 'test'", resp.Query)
    }
    // We don't assert result count — external service may be down
    // Smoke test only verifies: no panic, returns a response, query is preserved
}
```

### 6.7 — E2E (End-to-End) Tests
*Test the entire system as a user would experience it.*

- [ ] E2E tests use the **real binary or running server** — no mocks
- [ ] Tests simulate real user workflows (CLI commands, HTTP requests, UI interactions)
- [ ] Tests are **independent** — each test can run alone, in any order
- [ ] Tests clean up after themselves
- [ ] Flaky E2E tests are quarantined immediately — never ignored
- [ ] E2E tests run in CI but NOT on every commit (too slow) — on PR and nightly

**E2E for CLI applications:**
```go
func TestE2E_CLISearch(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test in short mode")
    }

    cmd := exec.Command("./searcheng", "search", "golang", "--providers=ddg", "--max-results=5")
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("CLI failed: %v\nOutput: %s", err, output)
    }

    out := string(output)
    if !strings.Contains(out, "Searching for: golang") {
        t.Error("missing search header")
    }
    if !strings.Contains(out, "DuckDuckGo") {
        t.Error("missing provider attribution")
    }
    if !strings.Contains(out, "results in") {
        t.Error("missing results summary")
    }
}
```

**E2E for REST API:**
```go
func TestE2E_APISearchFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test in short mode")
    }

    // Start real server
    cmd := exec.Command("./searcheng", "serve", "--port=19876")
    cmd.Start()
    defer cmd.Process.Kill()
    time.Sleep(2 * time.Second)

    // Hit real endpoint
    resp, err := http.Get("http://localhost:19876/search?q=test")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    if result["query"] != "test" {
        t.Errorf("query = %v, want 'test'", result["query"])
    }
    if results, ok := result["results"].([]interface{}); !ok || len(results) == 0 {
        t.Error("expected results array with data")
    }
}
```

**E2E for web UI (Playwright/Cypress):**
```typescript
// Example: E2E with Playwright (if a frontend is added later)
test('search flow shows results', async ({ page }) => {
  await page.goto('/');
  await page.fill('[data-testid="search-input"]', 'golang tutorial');
  await page.click('[data-testid="search-button"]');

  // Wait for results
  await expect(page.locator('[data-testid="result-item"]')).toHaveCount({ minimum: 1 });

  // Verify result structure
  const firstResult = page.locator('[data-testid="result-item"]').first();
  await expect(firstResult.locator('.result-title')).not.toBeEmpty();
  await expect(firstResult.locator('.result-url')).toContainText('http');
  await expect(firstResult.locator('.result-source')).not.toBeEmpty();
});

test('empty search shows error', async ({ page }) => {
  await page.goto('/');
  await page.click('[data-testid="search-button"]');
  await expect(page.locator('[data-testid="error-message"]')).toBeVisible();
});
```

### 6.8 — Contract Tests
*Verify that providers return data in the expected format.*

- [ ] Each provider has a contract test with a **recorded fixture** (golden file)
- [ ] Contract tests detect when a provider's HTML structure changes
- [ ] Contract tests run fast (no network) — parse local HTML files
- [ ] When a contract test fails, update the parser AND the fixture

```go
// Contract test: "DuckDuckGo HTML has the structure we expect"
func TestContract_DuckDuckGo_HTMLStructure(t *testing.T) {
    fixture, _ := os.ReadFile("testdata/duckduckgo_response.html")
    ddg := &DuckDuckGo{}
    results, err := ddg.parse(bytes.NewReader(fixture))
    if err != nil {
        t.Fatal(err)
    }

    // Contract: DDG returns at least 5 results
    if len(results) < 5 {
        t.Errorf("expected >= 5 results, got %d (HTML structure may have changed)", len(results))
    }

    // Contract: every result has title, URL, source
    for i, r := range results {
        if r.Title == "" {
            t.Errorf("result[%d]: empty title", i)
        }
        if r.URL == "" || !strings.HasPrefix(r.URL, "http") {
            t.Errorf("result[%d]: invalid URL %q", i, r.URL)
        }
        if r.Source != "DuckDuckGo" {
            t.Errorf("result[%d]: source = %q, want DuckDuckGo", i, r.Source)
        }
    }
}
```

### 6.9 — Performance / Load Tests
*Verify the system handles expected load without degradation.*

- [ ] Benchmark critical paths: search aggregation, HTML parsing, ranking
- [ ] Set performance budgets: "search must return in < 3s for 95th percentile"
- [ ] Load tests simulate concurrent users (k6, vegeta, hey, ab)
- [ ] No memory leaks under sustained load

```go
// Go benchmarks
func BenchmarkNormalizeURL(b *testing.B) {
    for i := 0; i < b.N; i++ {
        normalizeURL("https://EXAMPLE.COM/path/to/page/?q=test#section")
    }
}

func BenchmarkMergeAndRank_100Results(b *testing.B) {
    eng := &Engine{MaxResults: 50}
    providers := generateMockProviderResults(3, 100) // 3 providers, 100 results each
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        eng.mergeAndRank(providers)
    }
}
```

```bash
# Load test with k6
k6 run --vus 50 --duration 30s - <<'EOF'
import http from 'k6/http';
import { check } from 'k6';

export default function () {
  const res = http.get('http://localhost:8080/search?q=test');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 3s': (r) => r.timings.duration < 3000,
    'has results': (r) => JSON.parse(r.body).results.length > 0,
  });
}
EOF
```

### 6.10 — Test Execution Order in CI

```
CI Pipeline:
┌─────────────────────────────────────────────────────────────┐
│ 1. STATIC ANALYSIS  (< 30s)                                │
│    type-check → lint → format → vet                        │
├─────────────────────────────────────────────────────────────┤
│ 2. UNIT TESTS  (< 2min)                                    │
│    go test -short -race ./...                               │
│    pytest -m "not slow" / vitest --run                      │
├─────────────────────────────────────────────────────────────┤
│ 3. SMOKE TESTS  (< 30s)                                    │
│    Build binary → health check → basic search              │
├─────────────────────────────────────────────────────────────┤
│ 4. INTEGRATION TESTS  (< 5min)                             │
│    go test -run Integration ./...                           │
│    Contract tests with fixtures                             │
├─────────────────────────────────────────────────────────────┤
│ 5. E2E TESTS  (< 10min, PR + nightly only)                │
│    Full CLI flow, full API flow                             │
│    Playwright/Cypress if UI exists                          │
├─────────────────────────────────────────────────────────────┤
│ 6. PERFORMANCE TESTS  (nightly only)                       │
│    Benchmarks, load tests, memory profiling                │
└─────────────────────────────────────────────────────────────┘

FAIL FAST: If step N fails, steps N+1..6 are skipped.
```

### 6.11 — Test Anti-Patterns

| Anti-Pattern | Why It's Bad | Correct Approach |
|---|---|---|
| Testing implementation, not behavior | Breaks on every refactor | Assert on outputs and side effects |
| Shared mutable state between tests | Order-dependent failures, flakiness | Each test sets up its own state |
| Sleeping in tests (`time.Sleep(5s)`) | Slow and still flaky | Use polling, channels, or `Eventually()` |
| Ignoring flaky tests | Erosion of trust in the suite | Quarantine and fix within 48h |
| 100% code coverage as a goal | Leads to useless tests | Cover behavior, not lines |
| No tests on error paths | Bugs hide in error handling | Test every `if err != nil` branch |
| Giant test functions (100+ lines) | Hard to debug failures | One assertion per test, use table-driven tests |
| Mocking everything | Tests prove nothing | Mock boundaries only, test real logic |
| Tests that depend on network | Flaky in CI, impossible offline | Use fixtures, recorded responses, test servers |
| Copy-pasting test code | Maintenance nightmare | Use test helpers, table-driven tests, factories |

**Table-driven tests (Go idiom):**
```go
func TestNormalizeURL(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"strips fragment", "https://a.com/page#top", "https://a.com/page"},
        {"strips trailing slash", "https://a.com/page/", "https://a.com/page"},
        {"lowercases host", "https://A.COM/Page", "https://a.com/Page"},
        {"handles empty", "", ""},
        {"preserves query params", "https://a.com?q=1", "https://a.com?q=1"},
        {"handles malformed URL", "not-a-url", "not-a-url"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := normalizeURL(tt.input)
            if got != tt.want {
                t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

**Test coverage target:** Every new public function should have at least one happy path and one failure path test. Aim for meaningful coverage (critical paths, edge cases, error handling) — not a percentage metric.

---

## GATE 7 — Security

- [ ] HTTP endpoints: authentication middleware always present
- [ ] Input validation on every public entry point
- [ ] No SQL injection risk (parameterized queries, ORMs properly used)
- [ ] Rate limiting on public endpoints
- [ ] No sensitive data in logs
- [ ] No secrets in source code or git history
- [ ] CSRF protection on state-changing endpoints (if web)
- [ ] `withApiKeyAuth` or equivalent on webhook/callback endpoints

---

## GATE 8 — Architecture

- [ ] No cross-layer imports (frontend ← server internals)
- [ ] Domain boundaries respected (no domain A importing domain B's internals)
- [ ] Shared types in shared packages, not duplicated
- [ ] Generated files untouched — never manually edited (`_generated/`, `prisma/generated/`, etc.)
- [ ] No circular dependencies introduced
- [ ] New modules follow existing folder/file conventions

---

## GATE 9 — PR Hygiene

- [ ] Rebased on latest main before opening
- [ ] Schema has no duplicate fields after rebase
- [ ] No merge conflicts
- [ ] CI passing (all checks green)
- [ ] PR is focused — one concern only
- [ ] PR has fewer than 15 files (if larger, split by layer)
- [ ] No tool configuration files committed (`.claude/`, `.cursor/`, IDE files)
- [ ] Conventional commit format: `type(scope): description`
- [ ] PR links to the original issue

---

## Review Writing Standards

When writing code review comments, every finding follows this format:

```
**Problem**: [file:line] — specific description of what is wrong
**Why it matters**: [real consequence — not theory]
**Fix**:
  // Before (wrong)
  [problematic code]

  // After (correct)
  [fixed code with explanation]
```

### Review priority order:
1. **Security** — auth bypass, data exposure, injection, secrets
2. **Correctness** — bugs, wrong logic, edge cases, broken contracts
3. **i18n** — hardcoded strings, missing locales
4. **Conventions** — project wrappers, error codes, result patterns
5. **UI/UX** — design system, components, responsiveness
6. **Clean code** — dead code, imports, naming

### Review tone rules:
- Be specific, never vague ("change X to Y because Z" — not "consider improving")
- Always show the fix — never just point at the problem
- Reference existing patterns when suggesting alternatives ("like it's done in `path/to/file.ts`")
- Be assertive — if it's a problem, say so directly
- Be respectful — the goal is better code, not blame

---

## AI Review Confidence Scoring

When using AI code review tools, prioritize findings by confidence:

| Score | Action |
|-------|--------|
| > 0.75 | Fix before merge — almost certainly a real problem |
| 0.65–0.75 | Investigate — likely worth fixing |
| < 0.65 | Review with human judgment — may be false positive |

Common high-confidence issues:
- Cross-layer imports (0.84+)
- Missing discriminated union in schemas (0.96)
- Unbounded database queries (0.91)
- Missing `id` field on entities (0.76)
- `any` type in validators (0.66+)

---

## Pre-PR Final Checklist

Run this as the very last step before clicking "Open PR":

```
[ ] i18n: all locales updated, no orphan keys, no hardcoded strings
[ ] Backend: validators present, error enums used, queries bounded
[ ] Frontend: wrappers used, no cross-layer imports, typed routes
[ ] UI: design system used, loading/error/empty states present
[ ] Code: no dead code, no any, no console.log, no duplicate imports
[ ] Tests: type-check + lint + format + all tests pass
[ ] Security: auth on endpoints, no secrets, input validated
[ ] Architecture: no cross-layer, no generated file edits
[ ] PR: rebased, CI green, <15 files, focused, linked to issue
```

**Rule:** If you cannot check every box, you are not ready to open the PR.

---

## Common Failure Patterns (learned from real PRs)

### Stable key overwritten by display value
```typescript
// WRONG — overwrites the stable key used for lookups
items.map(item => ({ ...item, name: getDisplayLabel(item.name) }))

// CORRECT — keeps stable key, adds display field
items.map(item => ({ ...item, label: getDisplayLabel(item.name) }))
```

### Schema duplicate after rebase
```typescript
// Happens when both branches add the same field
// Git doesn't detect this as a conflict — different ordering
// Fix: always inspect schema files after rebase
```

### Bypassing project hooks/wrappers
```typescript
// WRONG — bypasses project error handling and auth
import { useMutation } from 'some-library/react';

// CORRECT — uses project wrapper with cross-cutting logic
import { useFunction } from '@/shared/hooks/useFunction';
```

### Unbounded database query
```typescript
// WRONG — will OOM in production
const allRecords = await db.collection.findAll();

// CORRECT — always paginate
const records = await db.collection.find().limit(100);
```

### Hardcoded error string
```typescript
// WRONG — untranslatable, unmaintainable
throw new Error("user_not_found_please_try_again");

// CORRECT — typed, translatable, consistent
throw new AppError(ErrorCode.USER_NOT_FOUND);
```

### Generated file manually edited
```bash
# WRONG — edit _generated/api.d.ts directly
# CORRECT — regenerate:
# bunx convex dev / prisma generate / graphql-codegen
# Then commit the regenerated output
```
