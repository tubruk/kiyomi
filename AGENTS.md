# AGENTS.md — Repository Maintenance & Guidelines for AI Agents

This document contains operational conventions, repository layout maps, and verification standards for AI agents working on the **Kiyomi** codebase.

---

## Main Branch Acceptance Criteria

All contributions and feature branches merged into the `main` branch must meet the following criteria:
- **No Private Data**: No leaked credentials, passwords, private keys, or absolute local filesystem/personal paths in tracked files.
- **Verification**: Unit tests, linters, and end-to-end (E2E) tests must pass cleanly.
- **Squash Merges**: When merging a single-goal feature branch, squash merging is highly preferred to drop intermediary commits and keep history clean.
- **Irrelevant Files**: Changes must not touch irrelevant Cucumber `.feature` files or unrelated test files.
- **Explicit Reviews**: Any explicit code or architectural review request must actively consult relevant Golang, backend, or frontend skills/guidelines for project best practices.

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

---

## 7. Repository Hygiene — No Personal Data in Tracked Files

AI agents MUST NOT introduce absolute local filesystem paths, personal usernames, home-directory references, machine-specific identifiers, or absolute `file://` URLs into any tracked file (source, docs, configs, skills, comments, test fixtures, or generated artifacts that end up in git).

### Forbidden Patterns in Tracked Files
- Absolute home paths: `/Users/<name>/...`, `/home/<name>/...`, `C:\Users\<name>\...`, `~/...` (when expanded).
- Absolute `file://` URLs of any kind (`file:///Users/...`, `file:///home/...`, etc.).
- Hardcoded local usernames, hostnames, MAC addresses, internal IPs (RFC1918 ranges), personal email addresses.
- Tooling output that embeds local paths (e.g., V8/Playwright coverage reports with `file://` sourcemaps) MUST be excluded via `.gitignore` (or `.dockerignore`) and never committed.

### Required Practices
- **Docs and skills**: reference repo files via **repo-relative paths** (e.g., `./pkg/foo/bar.go` or `pkg/foo/bar.go`), never `file://` URLs or absolute paths. If a link target is required, use a relative path or a public URL (GitHub blob URL is acceptable).
- **Code, comments, fixtures, generated artifacts**: never embed local absolute paths. Use placeholders (`$HOME`, `<repo-root>`, `os.UserHomeDir()`) when paths are needed at runtime.
- **Before committing**: run `git diff --cached | grep -nE '/Users/|/home/|file:///'` and remove any hits.
- **Pre-merge / pre-push gate**: `git grep -nE '/Users/|<your-username>|file:///'` on all tracked files MUST return zero results.
- **Test artifacts**: coverage reports (`.out`, Playwright `test-results/`, V8 JSON), IDE scratch dirs, `.env*`, and local DB files MUST be listed in `.gitignore` and `.dockerignore`.

---

## 8. Documentation Guidelines & Standards

To keep the repository clean, consistent, and maintainable, all documentation files under `docs/` must adhere to the following standards:

### File Naming Conventions
- **Lower-case Snake_Case**: All documentation files and directory names must use lower-case `snake_case` (underscores) rather than dashes or mixed-case.
  - **Correct**: `provider_plugin_architecture.md`, `plugin_developer/`, `_index.md`
  - **Incorrect**: `provider-plugin-architecture.md`, `plugin-developer/`, `README.md`
- **Entry Files**: All entry / index documentation files under `docs/` must be named `_index.md` instead of `README.md`.
- **Extension**: Use the `.md` extension for all markdown files.

### Relative Linking
- **Repository-Relative Links**: When referencing other files, source code, or documentation in the repository, always use repository-relative links (e.g. `../design/api.md` or `pkg/provider/sdk/sdk.go`).
- **No Local Paths or file:// URLs**: Never include absolute local file paths, personal directories, or `file://` URLs in any tracked documentation or codebase comments.

### Design Content Integrity
- **Conceptual Design**: Design documents (under `docs/design/`) must focus on specifications, schemas, protocols, and conceptual models, remaining completely agnostic of temporary implementation phases, tasks, or backlog plans.
