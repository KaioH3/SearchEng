# CLAUDE_WORKFLOW.md
# Master Development Workflow — Task to Perfect PR

> Language-agnostic. Framework-agnostic. Built from real PR review patterns.
> Use this file as context in every Claude Code session.

---

## Core Philosophy

| Principle | Rule |
|-----------|------|
| **Plan before code** | Never write a line without understanding the blast radius |
| **Reuse before create** | Always search for existing patterns first |
| **Small PRs** | One concern per PR. If >15 files, split by layer |
| **Atomic commits** | Something works? Commit it. Now. |
| **Think WHY not WHAT** | Trace upstream/downstream — understand consequences |
| **Backpropagate** | After every change, verify nothing broke above or below |

---

## Step 1 — Intake

Parse the task and extract requirements before touching any code.

- If GitHub issue: read the full issue, labels, linked PRs, comments
- Extract: what needs to be done, acceptance criteria, affected domains
- Classify: `feat` / `fix` / `refactor` / `docs` / `chore`
- Estimate scope: how many files? Which layers? Any migrations?

**Red flags at intake:**
- Vague requirements → clarify before starting
- Multiple unrelated concerns in one task → split into separate tasks
- "Quick fix" that touches core infrastructure → plan carefully

---

## Step 2 — Context Loading

Load project knowledge before any code. This is non-negotiable.

```
1. Read the project's main rules file (CLAUDE.md / .cursorrules / CONTRIBUTING.md)
2. Read any memory/decision log (MEMORY.md / ADRs / decision records)
3. Read architecture docs if touching a new domain
4. If UI task → check the design system / component library first
5. Check the schema / data model for the affected domain
```

**Key files to always locate:**
- Database schema / data model
- Shared type definitions
- Error code enums / constants
- Auth middleware / guards
- Test utilities / fixtures

---

## Step 3 — Plan (Enter Plan Mode)

Design before implementing. Present a written plan and validate it.

### Exploration checklist before writing the plan:
- [ ] Search for existing similar implementations ("how is this done elsewhere?")
- [ ] Map ALL files that will change: backend + frontend + locales + tests + migrations
- [ ] Identify who depends on what you're changing (downstream impact)
- [ ] Identify what you depend on (upstream contracts)
- [ ] Confirm the approach has the smallest blast radius possible

### Plan structure (always include these sections):

```markdown
## Context
- What: [clear description]
- Why: [motivation, issue link]
- Domain(s): [which layers are touched]

## Files
- [Complete list of every file to create / modify / delete]
- Include: locales, tests, migrations, config files

## Approach
- How it will be implemented
- Reference existing patterns: "following the pattern in path/to/file"
- Explain non-obvious decisions

## Risks
- What could break
- Edge cases
- Performance concerns

## Verification
- How to confirm it works
- Which tests to run
- Manual test steps
```

### Mental pre-flight before executing the plan:
1. Does the plan cover all i18n strings (if applicable)?
2. Are mutations/writes going through the correct abstraction layer?
3. Are there any cross-layer import violations?
4. Are existing components/functions reused?
5. Will the PR have fewer than 15 files? If not, split it.
6. Is there any dead code being left behind?

---

## Step 3.5 — Test Strategy (Define BEFORE implementing)

Before writing any production code, define your testing approach. This is the bridge between planning and implementation.

### Choose your methodology per feature type:

| Feature Type | Primary Methodology | Why |
|---|---|---|
| **New domain logic** (algorithms, ranking, scoring) | **TDD** | Pure functions → fast feedback, forces clean interfaces |
| **User-facing behavior** (search flow, API responses) | **BDD** | Stakeholder-readable scenarios, catches requirement gaps |
| **New bounded context / module** | **DDD testing** | Respects domain boundaries, tests aggregates through root |
| **Infrastructure** (new provider, HTTP client) | **Contract tests** + fixtures | Decouples from network, detects upstream HTML changes |
| **Bug fix** | **TDD** (regression-first) | Write failing test that reproduces bug → fix → never regresses |
| **Refactor** | **Characterization tests** | Capture current behavior first → refactor safely |

### TDD decision: when to use strict TDD

```
Is the logic pure (no I/O, no side effects)?
├── YES → STRICT TDD (Red → Green → Refactor)
│         Examples: URL normalization, ranking algorithm, score calculation,
│                   deduplication, input validation, string parsing
└── NO → Does it interact with external systems?
    ├── YES → Write CONTRACT TESTS with fixtures first, then implement
    │         Examples: HTML scraping parsers, API response handling
    └── NO → Write INTEGRATION TEST outline first, implement, verify
              Examples: HTTP server handlers, CLI command flow
```

### BDD decision: when to write Gherkin scenarios

```
Does the feature have user-visible behavior?
├── YES → Write Given/When/Then BEFORE code
│   ├── Is there a non-technical stakeholder?
│   │   ├── YES → Full Gherkin (.feature files) with automation
│   │   └── NO  → BDD-style test names: "should return merged results when both providers respond"
│   └── Map each acceptance criterion to at least one scenario
└── NO (internal refactor, performance) → Skip BDD, use TDD or benchmarks
```

### DDD test boundaries

```
For each Bounded Context, define:
1. DOMAIN tests     — pure logic, no I/O, test via aggregate root
2. APPLICATION tests — use cases with mocked ports (repositories, providers)
3. INFRASTRUCTURE tests — real adapters with fixtures or test servers
4. Never test across bounded context boundaries — use integration tests for that
```

### Test file mapping (add to your plan)

For every file in your plan's "Files" section, map its corresponding test:

```markdown
## Files + Tests
| Production File          | Test File                          | Test Type     |
|--------------------------|------------------------------------|---------------|
| engine/engine.go         | engine/engine_test.go              | Unit (TDD)    |
| engine/duckduckgo.go     | engine/duckduckgo_test.go          | Contract      |
| engine/google.go         | engine/google_test.go              | Contract      |
| api/server.go            | api/server_test.go                 | Integration   |
| main.go                  | e2e/cli_test.go                    | E2E           |
```

**Rule:** If a production file has no corresponding test file in your plan, justify why — or add the test.

---

## Step 4 — Implement

Execute the approved plan. For EVERY change:

### Backend
- [ ] Every API endpoint/function has input validation (Zod / class-validator / Pydantic / etc.)
- [ ] Errors use an enum or typed error system — never raw strings
- [ ] Database queries are bounded — never fetch unlimited rows
- [ ] Auth/permission checks on every protected operation
- [ ] HTTP endpoints have authentication middleware
- [ ] Retry logic only on transient errors (5xx), never on 4xx
- [ ] No secrets hardcoded — use environment variables

### Frontend
- [ ] All user-facing strings use the i18n system (`t('key')`) — never hardcoded
- [ ] Loading states for every async operation
- [ ] Error states handled and displayed
- [ ] Use existing UI components — don't recreate
- [ ] Styling via the design system (utility classes / theme tokens) — never inline styles
- [ ] Scrollable containers for long lists (mobile doesn't scroll automatically)
- [ ] Responsive — works on all target screen sizes
- [ ] Typed routes / navigation — no string-based magic paths

### Cross-cutting
- [ ] Shared types live in a shared package — not duplicated between layers
- [ ] Frontend does NOT import from server internals (except generated types)
- [ ] No `any` types — explicit or inferred
- [ ] No `console.log` in production code

### Commit discipline
```
# Pattern: one logical change = one commit
git add [specific files for this change]
git commit -m "type(scope): what changed"

# Types: feat | fix | refactor | style | test | docs | chore
# Scope: the domain/module (auth, payments, user-profile, etc.)
```

---

## Step 5 — Backpropagate (Enhance)

After implementing, review impacts you may have missed.

**Questions to answer:**
- Did changing this function break any caller?
- Did changing this schema break any query or migration?
- Did adding this component duplicate an existing one?
- Did removing this field leave orphan references?
- Is any code now unreachable / dead?
- Are all 3+ locales updated if strings changed?
- Do all existing tests still pass conceptually with the new behavior?

**Technique — trace the call graph:**
1. Find everything that calls what you changed
2. Find everything that what you changed calls
3. Verify contracts (inputs/outputs) are preserved at every layer
4. Check for side effects in mutations / writes

---

## Step 6 — Validate

Run the full validation suite. Every item must pass.

### 6a — Static Analysis (blocks everything else)
```bash
# Type safety
go vet ./...                            # Go
tsc --noEmit                            # TypeScript
mypy --strict .                         # Python
cargo check                             # Rust

# Linting
golangci-lint run ./...                 # Go
eslint . --ext .ts,.tsx                 # TypeScript
ruff check .                            # Python

# Formatting
gofmt -l .                              # Go (list unformatted files)
prettier --check .                      # TypeScript
black --check .                         # Python
```

### 6b — Test Pyramid (run bottom to top, fail fast)

```
                    ╱╲
                   ╱ E2E ╲         ← Slow, expensive, run on PR/nightly
                  ╱────────╲         Full system: CLI, HTTP, browser
                 ╱ Integration╲    ← Medium speed, run on every push
                ╱──────────────╲     Real adapters: HTTP handlers, parsers + fixtures
               ╱  Contract Tests  ╲  ← Fast, run on every commit
              ╱────────────────────╲   Provider response format validation
             ╱     Unit Tests       ╲← Fastest, run on every save
            ╱────────────────────────╲ Pure domain logic, mocked boundaries
           ╱    Static Analysis       ╲← Instant, run on every keystroke
          ╱────────────────────────────╲ Types, lint, format
```

### 6c — Execution commands by language

**Go:**
```bash
# Fast feedback (every save)
go vet ./... && go test -short -race -count=1 ./...

# Full suite (every commit)
go test -race -count=1 ./...

# With coverage
go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Benchmarks (on demand / nightly)
go test -bench=. -benchmem ./...

# E2E only
go test -run E2E -count=1 ./...

# BDD with godog
godog run features/
```

**TypeScript/JavaScript:**
```bash
# Unit + integration
vitest run                    # or jest --run
npm run test:unit
npm run test:integration

# E2E
npx playwright test           # browser E2E
npx cypress run               # alternative

# BDD with Cucumber
npx cucumber-js features/
```

**Python:**
```bash
# Unit
pytest -m "not slow and not e2e" --tb=short

# Integration
pytest -m "integration" --tb=short

# E2E
pytest -m "e2e" --tb=long

# BDD with Behave
behave features/

# BDD with pytest-bdd
pytest -m "bdd" --tb=short
```

### 6d — Test quality checklist

- [ ] All new public functions have tests (happy path + error path minimum)
- [ ] Tests are deterministic — run 3 times, same result every time
- [ ] No flaky tests introduced (check: `go test -count=10`)
- [ ] Tests run in isolation — no dependency on execution order
- [ ] Test names describe the scenario, not the method name
- [ ] Error messages are helpful — include expected vs actual with context
- [ ] No tests that test the framework or standard library (trust them)
- [ ] Mocks only at boundaries — never mock the thing you're testing

### 6e — When to write which test type

| Situation | Test Type | Command |
|---|---|---|
| New pure function | Unit (TDD) | `go test -run TestFunctionName` |
| New API endpoint | Integration | `go test -run TestEndpoint` |
| New HTML parser | Contract + fixture | `go test -run TestContract_Provider` |
| New CLI command | E2E | `go test -run TestE2E_CLI` |
| Bug report | Regression (TDD) | Write failing test → fix → `go test -run TestBugXXX` |
| Refactoring | Run ALL existing tests | `go test -race ./...` |
| Performance concern | Benchmark | `go test -bench BenchmarkX -benchmem` |
| Pre-deployment | Smoke | `./scripts/smoke-test.sh` |
| User story | BDD scenario | `godog run features/search.feature` |

### 6f — Coverage policy

| Metric | Target | Action if Below |
|---|---|---|
| Line coverage (new code) | > 80% | Add tests for uncovered branches |
| Branch coverage (new code) | > 70% | Add edge case tests |
| Critical path coverage | 100% | Non-negotiable — search, ranking, API |
| Overall project coverage | Trending up | Never merge code that decreases it |

**Coverage is a compass, not a destination.** High coverage with bad assertions is worse than low coverage with excellent tests. Prioritize meaningful tests over hitting a number.

Then run AI review (`/ai-review` or equivalent) and address all HIGH confidence findings.

*90 passing tests = confidence to deploy without fear.*

---

## Step 7 — PR

Create the pull request with precision.

### PR checklist:
- [ ] PR title: `type(scope): description` (conventional commit style)
- [ ] Links to the original issue
- [ ] Description covers: what changed, why, how to test
- [ ] No generated files committed with manual edits
- [ ] No `.env` / secrets / credentials in the diff
- [ ] No tool-specific files committed (`.claude/`, `.cursor/skills/`, IDE config)
- [ ] CI passes (type-check + lint + tests)
- [ ] PR is focused — one concern only

### PR description template:
```markdown
## Summary
- Bullet points of what was done

## Key Changes
| File/Area | Change | Impact |
|-----------|--------|--------|

## Test Plan
- Steps to manually verify the change works
- Edge cases tested
```

---

## Git Worktree — Parallel Feature Development

Work on multiple features simultaneously without stash chaos.

```bash
# Create isolated environments for each feature
git worktree add ../project-feature-a feat/feature-a
git worktree add ../project-feature-b feat/feature-b

# Now you have:
# project/              ← main (original)
# project-feature-a/   ← isolated branch
# project-feature-b/   ← isolated branch

# List active worktrees
git worktree list

# Clean up after merge
git worktree remove ../project-feature-a
```

Each worktree can run on a different port. No more context switching.

---

## Handling CI Failures

### Duplicate schema fields after rebase
```bash
git fetch origin main
git stash                     # save uncommitted work
git rebase origin/main        # rebase
git stash pop                 # restore
# Manually inspect schema for duplicate field names
```

### Generated files contaminated
```bash
# Files in _generated/, prisma/generated/, etc. are NEVER manually edited
# Restore to main version:
git checkout origin/main -- path/to/generated/file
```

### Duplicate imports
```bash
# Always check if a module is already imported before adding a new import line
# Merge into the existing import statement
```

---

## DDD (Domain-Driven Design) — Architecture for Testable Systems

### Why DDD matters for testing
DDD creates natural test boundaries. Each layer has a clear testing strategy. When domain logic is isolated from infrastructure, tests are fast, reliable, and meaningful.

### Layered Architecture

```
┌──────────────────────────────────────────────────┐
│                  PRESENTATION                     │
│         CLI (main.go) / REST API (api/)           │
│         Tests: E2E, smoke                         │
├──────────────────────────────────────────────────┤
│                  APPLICATION                      │
│         Use Cases / Orchestration                 │
│         Tests: Integration (mocked infra)         │
├──────────────────────────────────────────────────┤
│                    DOMAIN                         │
│         Entities, Value Objects, Services         │
│         Interfaces (Ports)                        │
│         Tests: Unit (TDD), BDD scenarios          │
├──────────────────────────────────────────────────┤
│                 INFRASTRUCTURE                    │
│         Providers (DDG, Google, Bing, Brave)      │
│         HTTP clients, caches, persistence         │
│         Tests: Contract, Integration (fixtures)   │
└──────────────────────────────────────────────────┘

DEPENDENCY RULE: arrows point INWARD only.
Infrastructure depends on Domain. Domain depends on NOTHING.
```

### Ports & Adapters (Hexagonal Architecture)

```
                    ┌─────────────┐
       CLI ────────►│             │◄──────── REST API
                    │   DOMAIN    │
  DuckDuckGo ──────►│  (Engine,   │◄──────── Brave API
                    │   Ranking,  │
     Google ────────►│   Results)  │◄──────── Cache
                    │             │
      Bing ─────────►│             │◄──────── Future DB
                    └─────────────┘

LEFT SIDE (Driving Adapters):     User triggers action → CLI, REST, gRPC
RIGHT SIDE (Driven Adapters):     Domain needs data → Providers, DB, Cache

PORTS:  Interfaces defined BY the domain (e.g., Provider interface)
ADAPTERS: Implementations that satisfy ports (e.g., DuckDuckGo struct)
```

### DDD building blocks and how to test each

| Building Block | What It Is | Test Strategy |
|---|---|---|
| **Entity** | Object with identity (e.g., SearchSession) | Test behavior through aggregate root |
| **Value Object** | Immutable, compared by value (e.g., Result, URL) | Test equality, immutability, validation |
| **Aggregate** | Cluster of entities with a root (e.g., Engine) | Test through root's public methods only |
| **Domain Service** | Stateless logic across entities (e.g., RankingService) | Unit test with TDD — pure functions |
| **Port** | Interface defined by domain (e.g., Provider) | Contract tests — verify adapter compliance |
| **Adapter** | Infrastructure implementation (e.g., DuckDuckGo) | Integration test with fixtures |
| **Application Service** | Orchestrates use case (e.g., SearchUseCase) | Integration test with mocked ports |
| **Domain Event** | Something that happened (e.g., SearchCompleted) | Assert event emission and payload |

### Applied to SearchEng

```
Domain Layer (engine/):
├── result.go          → Value Object: Result (Title, URL, Snippet, Source, Score)
├── provider.go        → Port: Provider interface
├── engine.go          → Aggregate Root: Engine (Search, mergeAndRank, normalizeURL)
│                        Domain Service: mergeAndRank (ranking algorithm)
│
Infrastructure Layer (engine/):
├── duckduckgo.go      → Adapter: implements Provider port
├── google.go          → Adapter: implements Provider port
├── bing.go            → Adapter: implements Provider port
├── brave.go           → Adapter: implements Provider port
│
Application Layer:
├── main.go            → Use Case orchestration (buildEngine, cmdSearch, cmdServe)
│
Presentation Layer:
├── main.go            → CLI (cmdSearch) + REST trigger (cmdServe)
├── api/server.go      → REST API adapter
│
Configuration:
├── config/config.go   → Infrastructure concern (environment)
```

### Ubiquitous Language

Define terms the whole team uses — in code, tests, docs, and conversations:

| Term | Meaning | NOT this |
|---|---|---|
| **Provider** | A search backend that returns results | "scraper", "fetcher", "source" (use consistently) |
| **Result** | A single search result with title, URL, snippet | "item", "entry", "link" |
| **Search** | Query sent to all providers in parallel | "request", "fetch" |
| **Merge** | Deduplicate + combine results from multiple providers | "aggregate", "join" |
| **Rank** | Score and sort results by relevance | "sort", "order" |
| **Provider failure** | A provider returned an error or timed out | "crash", "break" |

**Rule:** Code names MUST match the ubiquitous language. If the domain says "Provider", the code says `Provider`, tests say `TestProvider`, BDD says "Given the provider...".

---

## Anti-Patterns Reference

| Anti-Pattern | Correct Approach |
|---|---|
| Overwriting a stable key with a display value | Add a new field (`label`, `displayName`) — keep the key immutable |
| Raw library imports when the project has wrappers | Use project wrappers — they contain cross-cutting logic |
| Unbounded DB queries (`.findAll()`, `.collect()`) | Always paginate (`.take(limit)`) |
| Frontend importing server internals | Only import from generated types / shared packages |
| Hardcoded error strings | Use an ErrorCode enum |
| Hardcoded UI strings | Use the i18n system |
| Storing display text in the database | Store codes/keys, resolve to display text at render time |
| Large PRs mixing concerns | Split by layer and concern |
| Committing at the end of the day | Commit every logical step that works |
| Manual edits to generated files | Regenerate — never hand-edit |
| Writing tests AFTER the code | Write test first (TDD) or scenarios first (BDD) |
| Testing implementation details | Test behavior and outputs only |
| Mocking the thing under test | Mock only at boundaries (ports/adapters) |
| `time.Sleep` in tests | Use channels, polling, or `Eventually()` |
| Flaky tests left in CI | Quarantine and fix within 48h |
| Shared state between tests | Each test creates its own state from scratch |
| No tests on error paths | Every `if err != nil` branch needs a test |
| 100% coverage as a goal | Cover critical paths and edge cases meaningfully |
| Skipping smoke tests | Smoke tests catch "it won't even start" failures |
| E2E tests on every commit | E2E on PR + nightly only — too slow for every commit |
| Giant test functions (100+ lines) | Table-driven tests, one assertion per scenario |
| Testing across bounded contexts | Use integration tests at context boundaries |
