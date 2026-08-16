#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Check required binaries
if ! command -v go &>/dev/null; then
  echo "ERROR: 'go' binary not found. Please install Go (https://go.dev/) to build the backend server." >&2
  exit 1
fi

if ! command -v bun &>/dev/null; then
  echo "ERROR: 'bun' binary not found. Please install Bun (https://bun.sh/) to build the frontend and run tests." >&2
  exit 1
fi


echo "==> Building UI"
bun --cwd web build

echo "==> Copying UI into pkg/webui/dist for go:embed"
rm -rf pkg/webui/dist
rsync -a --copy-links web/dist/ pkg/webui/dist/

if [ "${SKIP_VERIFY:-}" != "true" ]; then
  echo "==> Building prod binary"
  go build -o ./bin/kiyomi ./cmd/kiyomi

  echo "==> Verifying mock absent from prod binary"
  if go tool nm ./bin/kiyomi 2>/dev/null | grep -q 'mock'; then
    echo "ERROR: mock provider leaked into prod binary" >&2
    exit 1
  fi
  echo "==> Prod binary clean (mock absent)"
fi

echo "==> Building e2e binary with coverage instrumentation"
go build -cover -coverpkg=./cmd/...,./internal/...,./pkg/... -tags e2e -o ./bin/kiyomi-e2e ./cmd/kiyomi

echo "==> Creating reports and test-results directories"
rm -rf e2e/test-results/coverage-go e2e/test-results/coverage-frontend-v8
mkdir -p e2e/reports e2e/test-results

if [ "${SKIP_BROWSER_INSTALL:-}" != "true" ]; then
  echo "==> Ensuring browser"
  bunx playwright install --with-deps chromium >/dev/null
fi

echo "==> Running e2e"
ARGS=()
for arg in "$@"; do
  if [[ "$arg" == docs/* ]]; then
    ARGS+=("../$arg")
  else
    ARGS+=("$arg")
  fi
done
cd e2e
EXIT_CODE=0
if [ ${#ARGS[@]} -gt 0 ]; then
  KIYOMI_E2E_BINARY="$(pwd)/../bin/kiyomi-e2e" bunx cucumber-js --config cucumber.config.cjs "${ARGS[@]}" || EXIT_CODE=$?
else
  KIYOMI_E2E_BINARY="$(pwd)/../bin/kiyomi-e2e" bunx cucumber-js --config cucumber.config.cjs || EXIT_CODE=$?
fi

echo "==> Post-processing E2E coverage"
if [ -d "test-results/coverage-go" ] && [ "$(ls -A test-results/coverage-go)" ]; then
  echo "==> Formatting Backend E2E Coverage"
  go tool covdata textfmt -i=test-results/coverage-go -o ../coverage-backend-e2e.out
fi

if [ -d "test-results/coverage-frontend-v8" ] && [ "$(ls -A test-results/coverage-frontend-v8)" ]; then
  echo "==> Formatting Frontend E2E Coverage"
  bunx c8 report --temp-directory test-results/coverage-frontend-v8 --reporter lcov --reports-dir ../web/coverage --exclude-after-remap true
fi

exit $EXIT_CODE




