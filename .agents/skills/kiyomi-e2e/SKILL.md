---
name: kiyomi-e2e
description: Standards, architecture, and workflows for writing, refactoring, maintaining, and reporting E2E tests in the Kiyomi project.
---

# Kiyomi E2E Testing Skill

This skill defines the operational standards and architectural requirements for maintaining, creating, and reporting End-to-End (E2E) tests in the Kiyomi codebase. All agents must read and follow this guide when tasked with writing features, fixing bugs, or updating E2E flows.

---

## 1. Architectural Rules

### 1.1 Complete State Isolation
* **Requirement**: Every scenario must run against a completely clean, isolated Go server instance.
* **Mechanism**: The E2E framework automatically spawns a fresh `kiyomi-e2e` server process and allocates a unique temporary directory (`KIYOMI_HOME`) for each scenario in the `Before` hook, and terminates it in the `After` hook.
* **Instruction**: Never write state cleanup logic (e.g. deleting entities) inside step definitions or hooks. Assume a 100% empty/clean environment at the start of every scenario.

### 1.2 Frontend-Only Interaction and Verification
* **Requirement**: Keep step definitions aligned with user interactions.
* **Given (Setup)**: You may use direct backend HTTP API calls (via `axios` or `fetch`) **only** during the `Given` setup phase to seed databases (e.g., seeding a manga to the library).
* **When (Actions)**: Must only use Playwright methods to interact with the DOM (e.g. `page.click()`, `page.fill()`). Do not trigger actions via API requests.
* **Then (Assertions)**: Must only check DOM states, routes, text visibility, or disabled properties. **Do not query backend APIs to assert success.**

---

## 2. Playwright Locator Guidelines

When interacting with the UI, prefer semantic locators and standard shadcn UI component structures:
* **Buttons**: Use text locators, e.g. `page.locator('button:has-text("Add to Library")')` or `page.locator('a:has-text("View in Library")')`.
* **Cards**: Use card content filters, e.g. `page.locator('[class*="bg-card"]').filter({ hasText: title }).first()`.
* **Inputs**: Target by placeholder or labels, e.g. `page.locator('input[placeholder="Search catalog..."]')`.
* **Waiting**: Always use `page.waitForFunction` to verify SPA client-side route transitions:
  ```typescript
  await page.waitForFunction(() => window.location.pathname.startsWith('/manga/'), { timeout: 10000 });
  ```

---

## 3. Running & Verifying Tests

E2E tests must always be run from the repository root:
* **Run all features**: `bun run e2e`
* **Run a single feature**: `bun run e2e docs/e2e/journeys/<name>.feature`
* **Run a specific scenario**: `bun run e2e docs/e2e/journeys/<name>.feature:<line>`

### Prerequisites & Pre-flight Checks
* **Frontend compilation**: Verify `bun run build` inside `web/` before running E2E tests.
* **Backend verification**: Ensure backend Go code compiles and tests pass (`go test -v ./...`).

---

## 4. Debugging & Failure Artifacts

When an E2E test scenario fails:
* **Reports**: Check generated HTML reports under `e2e/reports/`.
* **Screenshots & Traces**: Inspect failure screenshots and trace files in `e2e/test-results/`.
* **Server Output**: Inspect console logs for server startup or runtime errors.

---

## 5. Task Completion & Summary Requirements

Upon finishing any task involving E2E testing or feature implementation verified by E2E tests, the agent **MUST** present a summary that includes:

1. **E2E Test Execution Summary**:
   - List of executed feature files and scenarios.
   - Final pass/fail status and test output details.

2. **Codebase Under Test Changes**:
   - An itemized list of all modified, added, or deleted files in the application codebase under test (`cmd/`, `internal/`, `pkg/`, `web/`).
   - An itemized list of test infrastructure updates made to step definitions (`e2e/src/steps/`), feature specs (`docs/e2e/journeys/`), or fixtures (`docs/e2e/fixtures/`).

