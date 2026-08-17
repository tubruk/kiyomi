# E2E Test Plan — Kiyomi

> Isolated, BDD-style end-to-end tests covering every user journey in `docs/user-journey.md`. Runnable locally with one command. CI-ready behind a flag.

---

## 1. Goal & Non-Goals

**Goal:** One executable BDD spec per journey. Click the app like a user, assert on visuals + filesystem + DB. Run with `bun run e2e`. Optional CI later.

**Non-goals (v1):**
- Visual regression / pixel diff.
- Performance benchmarks.
- Load/stress testing.
- Provider compatibility (each external site is a separate concern; tested per-provider in unit tests already).

---

## 2. Stack

| Concern | Choice | Reason |
|---|---|---|
| Browser automation | **Playwright** | Real Chromium; needs to drive the React SPA. |
| BDD runner | **`@cucumber/cucumber`** (TS) | Gherkin ↔ Step defs. Runs on `bun`. |
| Language | **TypeScript** | Same toolchain as `web/`. |
| Server lifecycle | Spawned **Go binary** in a child process | No forks of `main.go`. |
| Mock provider | **In-process Go provider** (`pkg/provider/mock`) compiled with `-tags e2e` | No network. Deterministic. Stripped from prod binary. |
| Assertions | `@playwright/test`'s `expect` + `cucumber` assertions | Two layers: UI + state. |
| Reporting | Cucumber HTML + Playwright trace + screenshot on fail | Cheap, no SaaS. |

Why not Storybook / Vitest browser mode? Our journeys span UI + filesystem + DB + worker. A real browser is the only honest harness.

---

## 3. Architecture — One Server Per Scenario

```
┌────────────────────────── per scenario ──────────────────────────┐
│                                                                  │
│  tmpdir                                                           │
│  ├── library/            (seeded or empty fixtures)               │
│  └── cache/              (fresh)                                  │
│                                                                  │
│  child process: ./bin/kiyomi-e2e                               │
│  env: KIYOMI_HOME=tmpdir, KIYOMI_PORT=<worker-port>               │
│       KIYOMI_PROVIDER_CONFIG=tmpdir/providers.json               │
│       KIYOMI_MOCK_FIXTURES=fixtures/providers                   │
│                                                                  │
│  Playwright BrowserContext  (fresh, headless)                     │
│  └── 1 page → drives /reader/$id, /manga/$id, etc.               │
│                                                                  │
│  Cucumber World  (TS)  ←→  HTTP client  ←→  server REST          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

Each scenario gets:
- Fresh `KIYOMI_HOME` (mkdtemp).
- Fresh BrowserContext (cookies, storage cleared).
- Unique port: each parallel worker gets `4111+N` (N = worker index). No port discovery parsing needed.
- Mock provider pointed at JSON fixtures for that scenario.

Teardown: kill server, rm tmpdir, close context. No state leaks across scenarios.

---

## 4. Isolation Model

| Layer | What | How |
|---|---|---|
| Filesystem | Library / cache | `mkdtemp` per scenario in `os.tmpdir()/kiyomi-e2e-<uuid>` |
| Ports | HTTP listener | Each parallel worker claims `4111+N` (N = worker index). Range avoids collisions; no discovery parsing needed |
| Providers | External | Mock provider pointed at JSON fixtures serves everything |
| Browser | Cookies / localStorage | `browser.newContext()` per scenario |
| Time | `time.Now()` in worker | Short env TTLs (`KIYOMI_CACHE_PAGE_TTL=2s`) — real time, fast expiry. No fake clock. |
| Workers | Concurrency | `KIYOMI_GLOBAL_CONCURRENCY=2`, `KIYOMI_PROVIDER_CONCURRENCY=1` so queue order is predictable |
| IDs | Manga / chapter ULIDs | Mock provider uses stable IDs from JSON, not random |

**Cleanup invariant:** tests never write to the developer's real `KIYOMI_HOME`. The runner refuses to spawn the server if `KIYOMI_HOME` is unset OR points to anything under `$HOME/kiyomi` (safety belt).

---

## 5. Determinism

BDD is behavioral. We don't pin time — we assert on visible state.

**Plan:**
1. **Mock provider** pointed at scenario fixtures is used.
2. **No automatic scan on startup** — the library indexer is passive (no `IndexAll` called, no periodic scan registered). Nothing to disable.
3. **Mock provider IDs** are deterministic strings (`mock-manga-001`, `mock-chapter-001`).
4. **Cache TTL** where it matters: short TTLs via env (`KIYOMI_CACHE_PAGE_TTL=2s`, `KIYOMI_CACHE_SEARCH_TTL=1s`) — real time, fast expiry. No fake clock.
5. **Test waits** are explicit `await expect(...).toBeVisible()` — never `sleep(2000)`. Where the worker mutates state, the step polls the API until the expected state lands, with a timeout.

This stays honest: tests measure what the user sees, not what the wall clock says.

---

## 6. Directory Layout

```
kiyomi/
├── docs/
│   └── e2e/
│       ├── README.md                     # this file
│       ├── journeys/
│       │   ├── 01-explore-providers.feature
│       │   ├── 02-remote-manga-chapters.feature
│       │   ├── 03-add-manga.feature
│       │   ├── 04-library-manga-details.feature
│       │   ├── 05-read-manga.feature
│       │   ├── 06-download-chapters.feature
│       │   ├── 07-download-all.feature
│       │   ├── 08-organize.feature
│       │   ├── 09-explore.feature
│       │   ├── 10-switch-provider.feature
│       │   ├── 11-handle-orphans.feature
│       │   └── 12-refresh-chapters.feature
│       └── fixtures/
│           ├── providers/
│           │   ├── mock-a.json           # provider manifest
│           │   ├── catalog.json          # search results
│           │   ├── manga-alpha.json      # 24 chapters, 8 pages each
│           │   ├── manga-beta.json       # 5 chapters, varied page counts
│           │   └── cover-alpha.png       # 1×1 PNG, valid
│           └── library-seed/
│               ├── manga-x/              # used by Journey 2 (already in library)
│               └── manga-y/
│
├── cmd/kiyomi/
│   ├── main.go                         # func main — //go:build !e2e (excluded from e2e build)
│   └── main_e2e.go                   # //go:build e2e — e2e entry point, registers /api/v1/mock/ routes
│
├── pkg/provider/mock/                    # NEW — in-process provider, //go:build e2e
│   ├── provider.go
│   └── fixtures.go                       # catalog + manga JSON fixtures
│
├── e2e/                                   # NEW — test runner
│   ├── package.json
│   ├── tsconfig.json
│   ├── cucumber.config.js               # bun handles TS natively; .ts not accepted as --config
│   ├── playwright.config.ts             # traceDir: use.traceDir (not top-level)
│   ├── run.sh                            # one-command local runner
│   └── src/
│       ├── world.ts                      # custom World: server, ctx, home, port
│       ├── hooks.ts                      # Before/After — spawn, teardown
│       ├── pages/
│       │   ├── LibraryPage.ts
│       │   ├── MangaDetail.ts
│       │   ├── Reader.ts
│       │   ├── Explore.ts
│       │   └── Downloads.ts
│       ├── steps/
│       │   ├── common.steps.ts
│       │   ├── library.steps.ts
│       │   ├── reader.steps.ts
│       │   ├── download.steps.ts
│       │   ├── explore.steps.ts
│       │   ├── organize.steps.ts
│       │   ├── provider.steps.ts
│       │   └── orphan.steps.ts
│       └── support/
│           ├── server.ts                 # spawn child at worker port, poll GET /api/v1/library/manga until 200 (no /api/health exists)
│           ├── tmp.ts                     # mkdtemp KIYOMI_HOME, copy fixtures into tmp, track teardown
│           ├── ports.ts
│           └── fs.ts                     # assertions on disk + DB
│
└── .github/workflows/e2e.yml             # NEW — opt-in CI workflow
```

---

## 7. Feature Files ↔ User Journeys

Tags are user-readable, not numbered. Every scenario inside a feature file inherits the file's feature tag.

| Journey | Feature file | Tag |
|---|---|---|
| Explore providers | `01-explore-providers.feature` | `@explore-providers` |
| Remote manga chapters | `02-remote-manga-chapters.feature` | `@remote-manga-chapters` |
| 1. Add manga | `03-add-manga.feature` | `@add-manga` |
| 10. Library manga details | `04-library-manga-details.feature` | `@library-manga-details` |
| 2. Read manga | `05-read-manga.feature` | `@read-manga` |
| 3. Download chapters | `06-download-chapters.feature` | `@download-chapters` |
| 4. Download entire manga | `07-download-all.feature` | `@download-all` |
| 5. Organize | `08-organize.feature` | `@organize` |
| 6. Explore | `09-explore.feature` | `@explore` |
| 7. Switch provider | `10-switch-provider.feature` | `@switch-provider` |
| 8. Handle orphans | `11-handle-orphans.feature` | `@handle-orphans` |
| 9. Refresh chapter list | `12-refresh-chapters.feature` | `@refresh-chapters` |

Cross-cutting tags:
- `@smoke` — must-pass subset (one happy-path scenario per journey). Used by the `run-e2e` CI label and a fast local loop.
- `@slow` — scenarios that take >5s (e.g. bulk download). Skipped by default.

Each feature carries a `Background:` that:
1. Spawns a server with seeded library `manga-x` (if the journey needs prior state).
2. Navigates to the relevant screen. (No login step — Kiyomi has no auth.)

---

## 8. Mock Provider Spec — Test-Only Build

**Rule:** the mock provider must not ship in the production binary. Kiyomi is a self-hosted tool users `go install` — they should never see a `mock` provider in their UI.

**Mechanism:** Go build tags.

```go
// pkg/provider/mock/provider.go
//go:build e2e

package mock
```

A gated registration file wired into the provider registry:

```go
// pkg/provider/registry_e2e.go
//go:build e2e

package provider

import "github.com/tubruk/kiyomi/pkg/provider/mock"

func init() {
    Register(mock.New())
}
```

**Build commands:**
```bash
# Production binary — no mock, no overhead, no leak
go build -o bin/kiyomi ./cmd/kiyomi

# E2E binary — main_e2e.go compiled when -tags e2e is set
go build -tags e2e -o bin/kiyomi-e2e ./cmd/kiyomi
```

The `e2e/run.sh` script always builds `bin/kiyomi-e2e`. The `make build` target never does. There is no env flag that pulls the mock in at runtime — it's literally not in the binary.

**Capabilities exposed:**
- `Search(query)` → entries from per-scenario `catalog.json`.
- `GetManga(remoteID)` → metadata from `manga-<id>.json`.
- `GetChapters(remoteID)` → list of chapter IDs.
- `GetPages(remoteID, chapterID)` → list of page URLs (served by mock HTTP handler at `/api/v1/mock/<remoteID>/<chapterID>/<pageIndex>`).
- `GetImage(url)` → bytes from `cover-alpha.png` or generated 1×1 PNG.

**Mock HTTP handler:** `cmd/kiyomi/main_e2e.go` creates its own Echo instance, calls `handler.RegisterRoutes`, then appends a mock route group at `/api/v1/mock/`. This keeps mock page serving in-process — no network, no external dependency.

**Fixture path:** the runner sets `KIYOMI_MOCK_FIXTURES=fixtures/providers` before spawning the server. The mock resolves this relative to `KIYOMI_HOME` (the tmpdir). The runner copies `docs/e2e/fixtures/providers/` into `<tmpdir>/fixtures/providers/` before server start.

**Stable IDs:** everything prefixed `mock-` (`mock.alpha.chapter.007`, etc.) — makes step assertions unambiguous.

---

## 9. Step Definition Layers

Three reusable layers:

### Layer A — Pure DOM steps (`steps/common.steps.ts`)
```
Given I open {string}
When I click {string}
Then I see {string}
Then I see {string} in {locator}
```

### Layer B — Domain steps (`steps/library.steps.ts`, `steps/reader.steps.ts`, …)
Compose Layer A with the page objects. Each domain step is one user-meaningful action:
```
When I add the manga {string} from the search modal
When I read chapter {string}
Then the chapter progress is saved at page {int}
```

### Layer C — System assertions (`support/fs.ts`)
Read-only asserts against the live `KIYOMI_HOME` for that scenario:
```
Then the library contains {string}
Then the file {string} exists on disk
Then the DB shows {int} download jobs completed
```

Layer C is what catches the bugs Layer A can't see: e.g. "user sees green dot" but "no files on disk" → that's a regression.

---

## 10. Local Runner

`e2e/run.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# 1. Build both binaries.
echo "==> Building prod binary"
go build -o ./bin/kiyomi ./cmd/kiyomi

echo "==> Building e2e binary (mock provider compiled in)"
go build -tags e2e -o ./bin/kiyomi-e2e ./cmd/kiyomi

# 2. Verify mock is absent from prod build (sanity check).
# go tool nm is Go-native and works across macOS (Mach-O) and Linux (ELF).
if go tool nm ./bin/kiyomi 2>/dev/null | grep -q 'mock'; then
  echo "ERROR: mock provider leaked into prod binary" >&2
  exit 1
fi
echo "==> Prod binary clean (mock absent)"

# 3. Install Playwright browser if needed.
# --with-deps: Linux = system browser deps via apt; macOS = browser binary only.
echo "==> Ensuring browser"
bunx playwright install --with-deps chromium >/dev/null

# 4. Run cucumber.
# bun transpiles .js and .ts files natively; no ts-node needed.
# Playwright config (playwright.config.ts) is NOT auto-loaded by cucumber-js;
# hooks.ts imports it explicitly and applies config to the Playwright browser instance.
echo "==> Running e2e"
cd e2e
bunx cucumber-js --config cucumber.config.js --workers=2 --publish=none
```

> **macOS note:** `playwright install --with-deps` downloads the Chromium binary but does not install system packages (not needed on Darwin). On Linux CI the flag also pulls browser system deps.

> **Cucumber config:** `--config cucumber.config.js` is required — cucumber-js does not auto-detect `.ts` files. bun transpiles `.js` natively without ts-node. `cucumber.config.js` sets `formatters: [['html:reports/cucumber.html']]` so the HTML report lands at `e2e/reports/cucumber.html`.

User-facing commands:
```bash
bun run e2e                       # full suite
bun run e2e --tags @read-manga    # one journey
bun run e2e --tags @smoke         # fast smoke loop
bun run e2e --tags "@smoke and not @slow"   # default in CI
```

`package.json` (root) addition:
```json
"e2e": "bash e2e/run.sh",
"e2e:ui": "bash e2e/run.sh -- --tags @smoke"
```

`e2e/package.json` dev dependencies:
```json
"devDependencies": {
  "@cucumber/cucumber": "^11",
  "@cucumber/html-formatter": "^21",
  "@playwright/test": "^1.49",
  "typescript": "^5"
}
```

Parallel runs: cucumber-js + Playwright `--workers=2` per scenario. Server spawn is cheap (<300ms). Target: **all 9 journeys in <90s** on a laptop.

---

## 11. CI Enablement (Now Behind a Flag, Off by Default)

Add `.github/workflows/e2e.yml` but guarded by `workflow_dispatch` + a label `run-e2e` on the PR. **No trigger on push.** When ready to enforce, remove the guard.

```yaml
name: e2e
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    if: contains(github.event.pull_request.labels.*.name, 'run-e2e')
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v1
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: bun install
      - run: bash e2e/run.sh
      - if: failure()
        uses: actions/upload-artifact@v4
        with: { name: playwright-traces, path: test-results/ }
```

**Enable on CI later:** delete the `if:` line + the `workflow_dispatch` triggers. One PR.

---

## 12. Phased Rollout

| Phase | Deliverable | Acceptance |
|---|---|---|
| **0. Prereqs** | `pkg/provider/mock` (build tag `e2e`), `cmd/kiyomi/main_e2e.go` (e2e entry point with mock routes), `KIYOMI_WEB_DIR` already used for dev, health via `GET /api/v1/library/manga` (no /health endpoint), no automatic startup scan to disable, mock provider implements all `sdk` interfaces accepted by `Registry.Register` | `go build ./...` and `go build -tags e2e ./...` both compile; `go tool nm bin/kiyomi \| grep -q mock` fails; `go tool nm bin/kiyomi-e2e \| grep -q mock` passes |
| **1. Harness** | `e2e/` scaffold, `run.sh`, one trivial smoke scenario (`Then I see the empty library`) | `bun run e2e` boots server, opens browser, asserts DOM, tears down |
| **2. Journey 1 + 2** | Add manga + Read | Two `.feature` files, all green |
| **3. Journey 3 + 4** | Download chapters + bulk download | Worker assertions via `Layer C` |
| **4. Journey 5** | Organize (status, collections, tags, filters) | No filesystem asserts needed |
| **5. Journey 6** | Explore + provider-down | Mock provider fails on demand |
| **6. Journey 7 + 9** | Switch provider + refresh | Stable ID strategy must align with mock |
| **7. Journey 8** | Orphans | Requires provider fixture to remove chapters |
| **8. CI** | Wire workflow | PR with `run-e2e` label runs all 9 |

Each phase is a single PR. AGENTS.md mandates verifications: `go test ./...`, `bun run build`, and `bun run e2e` for any phase that touches the harness.

---

## 13. Open Decisions

1. **Auth.** Kiyomi has no auth by design — anyone who reaches the server can use it. Background step does not include login. Close: no action needed.
2. **WebSocket.** Codebase is HTTP/REST only — confirmed zero WebSocket usage across `internal/` and `web/src`. Close: no action needed.
3. **Screenshot on every step vs only on failure.** Default: only on failure, plus a `Then I screenshot {name}` step for ad-hoc captures.
4. **Re-using library directories between scenarios.** Avoided by architecture — all data access is via directory paths configured per handler. Close.
5. **Coverage of the **System Journeys** (Library Scan, Cache Eviction, Progress Sync) from `user-journey.md`.** Not user-visible. Cover indirectly via Layer C asserts in the journey scenarios. Promote to their own feature file if regressions bite.

---

## 14. Quick Reference

| Command | What |
|---|---|
| `bun run e2e` | Run all 9 journeys |
| `bun run e2e --tags @read-manga` | One journey |
| `bun run e2e --tags @smoke` | Smoke subset |
| `bun run e2e --tags "@smoke and not @slow"` | CI default |
| `e2e/reports/cucumber.html` | HTML report |
| `test-results/` | Playwright traces on failure |

A green `bun run e2e` is the contract: every documented journey works, from the user's eyes, end to end.
