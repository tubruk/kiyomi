# Backlog: Full-Stack Code Coverage & Codecov Multi-Flag Reporting

This backlog item outlines the strategy to measure, extract, and report code coverage across both the **Go Backend Server** and the **Vite React Web UI** during unit tests and E2E test runs (`bun run e2e`), uploading multi-flag reports to Codecov for separate and combined README badges.

---

## Objective

1. **Backend Coverage**: Measure line coverage for Go packages (`cmd/`, `internal/`, `pkg/`) using `go test -cover` for unit tests and Go 1.20+ binary coverage instrumentation (`-cover`) during E2E runs.
2. **Frontend Coverage**: Measure JS/TSX execution coverage for `web/src/` components during E2E Playwright test runs.
3. **Codecov Integration**: Configure `codecov.yml` with `backend` and `frontend` flags to generate:
   - **Backend Coverage Badge** (`?flag=backend`)
   - **Frontend Coverage Badge** (`?flag=frontend`)
   - **Combined Full-Stack Coverage Badge** (default unified graph)

---

## Technical Strategy

### 1. Go Backend Coverage Extraction (`-cover` & `GOCOVERDIR`)

#### A. Build Instrumente Binary
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

### 2. Frontend Web UI Coverage Extraction (Playwright V8)

#### A. Capture JS Coverage in Playwright (`e2e/src/hooks.ts`)
```typescript
Before(async function () {
  await this.page.coverage.startJSCoverage();
});

After(async function (scenario) {
  const coverage = await this.page.coverage.stopJSCoverage();
  // Filter for web/src frontend code only
  const appCoverage = coverage.filter(entry => entry.url.includes('/src/'));
  
  const targetDir = './test-results/coverage-frontend-v8';
  fs.mkdirSync(targetDir, { recursive: true });
  fs.writeFileSync(`${targetDir}/${scenario.picker.id}.json`, JSON.stringify(appCoverage));
});
```

#### B. Format into `lcov.info`
At the end of `e2e/run.sh`:
```bash
npx c8 report --temp-directory test-results/coverage-frontend-v8 --reporter lcov --reports-dir web/coverage
```

---

### 3. Codecov Multi-Flag Configuration (`codecov.yml`)

Create `codecov.yml`:
```yaml
coverage:
  status:
    project:
      default:
        target: auto
      backend:
        flags: [backend]
      frontend:
        flags: [frontend]

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
- name: Upload Backend Coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    files: ./coverage-backend.out
    flags: backend
    name: backend-coverage

- name: Upload Frontend Coverage to Codecov
  uses: codecov/codecov-action@v4
  with:
    files: ./web/coverage/lcov.info
    flags: frontend
    name: frontend-coverage
```

#### README Badges (`README.md`)
```markdown
[![Combined Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg)](https://codecov.io/gh/tubruk/kiyomi)
[![Backend Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg?flag=backend)](https://codecov.io/gh/tubruk/kiyomi/flags/backend)
[![Frontend Coverage](https://codecov.io/gh/tubruk/kiyomi/branch/main/graph/badge.svg?flag=frontend)](https://codecov.io/gh/tubruk/kiyomi/flags/frontend)
```
