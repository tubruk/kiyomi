# AGENTS.md — Repository Maintenance & Guidelines for AI Agents

This document contains operational conventions, repository layout maps, and verification standards for AI agents working on the **Kiyomi** codebase.

---

## 1. Project Structure Map

- `cmd/kiyomi/`: Main Go application entry point.
- `internal/`: Server application logic, API routing (`internal/api`), SQLite database storage (`internal/library`), configuration (`internal/config`), and built-in providers (`internal/provider`).
- `pkg/provider/sdk/`: Extension SDK interfaces (`MetadataProvider`, `ContentProvider`, `Tracker`) and HTTP utilities.
- `web/`: Kiyomi Web UI Vite SPA (React 18 + Tailwind CSS v4 + shadcn/ui + TanStack Router/Query). Served by the Go server at `/`. API routes live under `/api/v1/`.
- `docs/`: System documentation and developer guides.
  - `docs/user/`: Operator guides, user manuals, environment configurations, and setup specs.
  - `docs/design/`: Core architectural designs, specifications, schemas, protocols, and conceptual specs (must never mention implementation phases).
  - `docs/plugin_developer/`: Guides, quickstarts, and reference manuals for building external provider plugins.
  - `docs/backlogs/`: Features, specifications, or improvements that are planned/proposed but not yet implemented.
  - `docs/e2e/`: Guides, conventions, and setups for running Playwright-based end-to-end tests.

---

## 2. Git Commit Conventions

### Strict Scope Rule
- **At most ONE scope per conventional commit.**
  - **Correct**: `feat(api): add popular and latest endpoints`
  - **Correct**: `feat(web): add popular and latest catalog tabs`
  - **Incorrect**: `feat(api,web): add popular and latest endpoints`

### Brief & Imperative Subject Line
- Keep the commit message subject line brief (under 72 characters).
- Use lower-case after the type/scope prefix.
- Write in the imperative mood (e.g. `add`, `fix`, `refactor`, `update`, NOT `added` or `adding`).

### Recognized Commit Types
- `feat`: New feature or API endpoint
- `fix`: Bug fix
- `refactor`: Code restructuring without API contract changes
- `docs`: Documentation updates
- `test`: Adding or updating unit/integration tests
- `chore`: Build scripts, dependencies, or maintenance tasks

---

## 3. Web UI Operational Workflow

- **Location**: Kiyomi Web UI lives in `web/` and is served as a Vite SPA at `/`.
- **Shadcn CLI Workflow**:
  - Run component additions using `bunx --bun shadcn@latest add <component>` inside `web/`.
  - UI components live at `web/src/components/ui/<component>.tsx`.

### Client-Side Sorting & Filter Principle
- For non-paginated endpoints where full datasets are returned in a single HTTP payload (such as chapter lists), sorting and filtering MUST be performed client-side in memory using React `useMemo`.
- Avoid passing sorting parameters into TanStack Query keys when data is unpaginated, preventing unnecessary network re-fetches and query cache churn when users toggle sort directions or criteria.

---

## 4. Verification & Quality Assurance

- **Go Backend**: Always run `go test -v ./...` from root before declaring backend tasks complete.
- **Web Frontend**: Always verify the frontend build with `bun run build` inside `web/`.
- **Test Integrity**: Do not delete or disable unit tests to mask errors.

---

## 5. Subagent Operations & Complex Refactoring Guidelines

- **Delegation Strategy**: Prefer delegating non-trivial editing, implementation, multi-step, or large refactoring tasks to dedicated subagents using cheap, low-thinking models (e.g., `'flash'` or `'flash_lite'`) to minimize cost and keep context focused. This delegation strategy also applies when executing an approved plan—ensure the plan is complete, sufficient, and self-sustained to direct subagents without excessive back-and-forth communication.
- **Phased Execution**: Break complex refactoring into distinct, sequential phases (e.g., Phase 1: Core Models & Data Storage, Phase 2: API Endpoints & Handlers, Phase 3: Frontend UI Components & Integrations). Phased execution / phasing (such as 'Phase 1') is strictly a planning/coordination tool for execution. It must never leak into source code comments, commit messages, PR titles/descriptions, or design documents (which must remain conceptual and implementation-phase agnostic).
- **Self-Contained Prompts**: Write detailed, self-contained subagent prompts with clear task boundaries, target file paths, DB schema specs, and expected outputs.
- **Incremental Scope-Compliant Commits**: Require subagents to create atomic commits per phase, strictly respecting the single-scope conventional commit rule (e.g., `refactor(model)`, `refactor(library)`, `feat(api)`, `feat(web)`).
- **Mandatory Verification**: Every subagent MUST run `go test -v ./...` from root and `bun run build` inside `web/` before declaring a task complete.

---

## 6. Error Investigation & Diagnostic Protocol

- **Root Cause & Proposal First**: When the user reports a new runtime error, log snippet, or unexpected behavior, AI agents MUST investigate the codebase to identify the exact root cause and present a clear diagnosis along with a proposed solution to the user before applying fixes or mutating code.

### Error Logging & Inspection Invariants
- **Backend Logging**: HTTP logging middleware MUST only log failed requests (status >= 400 or handler errors) to keep terminal logs clean and free of 200 OK access noise. Provider error handlers MUST log explicit `slog.Error` messages with route URI, provider ID, status code, and underlying error cause.
- **Frontend Toast & Modal Inspection**: Frontend mutation error handlers MUST NOT dump raw stack traces or un-truncated HTML into toast messages. Render a concise preview in the toast paired with a "Details" action button that opens a manually dismissable Error Details Modal containing a scrollable monospace trace box and a "Copy Error" button.
