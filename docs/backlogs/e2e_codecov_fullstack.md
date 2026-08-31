# Backlog: Full-Stack Code Coverage & Codecov Multi-Flag Reporting

This backlog item outlines the strategy to measure, extract, and report code coverage across both the **Go Backend Server** and the **Vite React Web UI**, uploading multi-flag reports to Codecov for separate and combined README badges.

---

## Objective

1. **Backend Coverage**: Measure line coverage for Go packages (`cmd/`, `internal/`, `pkg/`) using `go test -cover` for unit tests and Go 1.20+ binary coverage instrumentation (`-cover`) during E2E runs.
2. **Frontend Coverage**: Measure JS/TSX execution coverage for `web/src/` components and utilities using Vitest + React Testing Library + `@vitest/coverage-v8` during CI (`ci.yml`). E2E Playwright V8 coverage is maintained as a local/CI artifact.
3. **Codecov Integration**: Configure `codecov.yml` with `backend` and `frontend` flags to generate:
   - **Backend Coverage Badge** (`?flag=backend`)
   - **Frontend Coverage Badge** (`?flag=frontend`)
   - **Combined Full-Stack Coverage Badge** (default unified graph)

---

## Technical Strategy

### 1. Go Backend Coverage Extraction (`-cover` & `GOCOVERDIR`)

#### A. Build Instrumented Binary
In `e2e/run.sh`:
```bash
go build -cover -coverpkg=./cmd/...,./internal/...,./pkg/... -o bin/kiyomi-e2e ./cmd/kiyomi
```

#### B. Set Environment Variable in E2E Hooks
In `e2e/src/hooks.ts`:
```typescript
const coverDir = path.join(__dirname, '../test-results/coverage-go');
fs.mkdirSync(coverDir, { recursive: true });
process.env.GOCOVERDIR = coverDir;
```

#### C. Format Coverage Output
At the end of `e2e/run.sh`:
```bash
# Merge unit test coverage & binary cover data
go tool covdata textfmt -i=test-results/coverage-go -o coverage-backend-e2e.out
```

---

### 2. Frontend Web UI Unit Test Coverage (Vitest & V8)

#### A. Unit and Component Tests
In `web/`:
```bash
bun run test:coverage
```
Vitest executes tests in `src/` against jsdom, utilizing `@vitest/coverage-v8` to output standard `lcov.info` reports to `web/coverage/lcov.info`.

#### B. E2E Coverage (Playwright V8 Artifacts)
During E2E runs (`e2e/run.sh`), Playwright records browser V8 coverage to `e2e/test-results/coverage-frontend-v8/` as a local inspection and troubleshooting artifact.

---

### 3. Codecov Multi-Flag Configuration (`codecov.yml`)

`codecov.yml`:
```yaml
coverage:
  status:
    project:
      default:
        target: 30%
      backend:
        flags: [backend]
      frontend:
        flags: [frontend]
    patch: off

flags:
  backend:
    paths:
      - internal/
      - pkg/
      - cmd/
    carryforward: true

  frontend:
    paths:
      - web/src/
    carryforward: true
```

---

### 4. CI Workflow & README Badges

#### GitHub Actions (`.github/workflows/ci.yml`)
```yaml
- name: Run Frontend Unit Tests with Coverage
  run: |
    cd web
    bun run test:coverage

- name: Upload Backend Unit Coverage to Codecov
  uses: codecov/codecov-action@v5
  with:
    token: ${{ secrets.CODECOV_TOKEN }}
    files: ./coverage-backend-unit.out
    flags: backend
    name: backend-unit-coverage
    fail_ci_if_error: false

- name: Upload Frontend Unit Coverage to Codecov
  uses: codecov/codecov-action@v5
  with:
    token: ${{ secrets.CODECOV_TOKEN }}
    files: ./web/coverage/lcov.info
    flags: frontend
    name: frontend-coverage
    fail_ci_if_error: false
```

#### README Badges (`README.md`)
```markdown
[![Combined Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg)](https://codecov.io/gh/tubruk/kiyomi)
[![Backend Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg?flag=backend)](https://codecov.io/gh/tubruk/kiyomi/flags/backend)
[![Frontend Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg?flag=frontend)](https://codecov.io/gh/tubruk/kiyomi/flags/frontend)
```

