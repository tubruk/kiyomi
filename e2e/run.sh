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

echo "==> Building prod binary"
go build -o ./bin/kiyomi ./cmd/kiyomi

echo "==> Building e2e binary (mock provider compiled in)"
go build -tags e2e -o ./bin/kiyomi-e2e ./cmd/kiyomi

echo "==> Verifying mock absent from prod binary"
if go tool nm ./bin/kiyomi 2>/dev/null | grep -q 'mock'; then
  echo "ERROR: mock provider leaked into prod binary" >&2
  exit 1
fi
echo "==> Prod binary clean (mock absent)"

echo "==> Creating reports and test-results directories"
mkdir -p e2e/reports e2e/test-results

echo "==> Ensuring browser"
bunx playwright install --with-deps chromium >/dev/null

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
if [ ${#ARGS[@]} -gt 0 ]; then
  KIYOMI_E2E_BINARY="$(pwd)/../bin/kiyomi-e2e" bunx cucumber-js --config cucumber.config.cjs "${ARGS[@]}"
else
  KIYOMI_E2E_BINARY="$(pwd)/../bin/kiyomi-e2e" bunx cucumber-js --config cucumber.config.cjs
fi



