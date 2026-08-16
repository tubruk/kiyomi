# Backlog: Pre-Baked CI Container Image for E2E speedup

This backlog item outlines the design for creating and maintaining a pre-baked Docker container image to act as the runner environment for Kiyomi E2E tests, avoiding browser downloads and runtime dependency installs on every CI execution.

---

## Objective

Reduce E2E workflow startup latency by replacing general-purpose GitHub runner execution with a pre-configured Docker container hosting:
*   Node/Bun runtime environment.
*   Go compiler toolchain.
*   Pre-downloaded Playwright Chromium browser binaries and system-level browser dependencies.

---

## Technical Strategy

1.  **Define Runner Dockerfile (`.github/runner.Dockerfile`)**:
    ```dockerfile
    FROM mcr.microsoft.com/playwright:v1.49.0-noble

    # Install Go
    RUN apt-get update && apt-get install -y golang-go rsync && rm -rf /var/lib/apt/lists/*
    
    # Install Bun
    RUN npm install -g bun
    ```

2.  **Workflow Registry Integration**:
    *   Build and push the runner image to GHCR (e.g. `ghcr.io/tubruk/kiyomi-ci-runner:latest`) periodically or on runner Dockerfile changes.
    *   Specify the container in `.github/workflows/ci.yml`:
        ```yaml
        e2e-tests:
          runs-on: ubuntu-latest
          container:
            image: ghcr.io/tubruk/kiyomi-ci-runner:latest
          steps:
            # Steps now skip browser installs
        ```
