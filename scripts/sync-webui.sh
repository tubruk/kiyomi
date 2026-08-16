#!/bin/bash
set -e

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
REPO_ROOT="$DIR/.."

# Clean pkg/webui/dist
rm -rf "$REPO_ROOT/pkg/webui/dist"
mkdir -p "$REPO_ROOT/pkg/webui/dist"

# Copy from web/dist
if [ -d "$REPO_ROOT/web/dist" ]; then
  cp -r "$REPO_ROOT/web/dist/"* "$REPO_ROOT/pkg/webui/dist/"
  echo "Synced web/dist to pkg/webui/dist"
else
  # Keep a placeholder index.html so go:embed doesn't fail
  echo "<h1>Kiyomi Placeholder</h1>" > "$REPO_ROOT/pkg/webui/dist/index.html"
  echo "Warning: web/dist does not exist, created placeholder index.html"
fi
