#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

BRANCH="task/BUG-007-01-cache-and-visual-polish"

echo "=== Packaging and Pushing ${BRANCH} ==="
git push origin "${BRANCH}"
echo "=== Pushed branch ${BRANCH} to origin ==="

echo ""
echo "=== Public Edge CDN Verification (https://goodai.co.nz) ==="
echo "--- Root / Response Headers ---"
curl -sI https://goodai.co.nz/ || true

echo "--- style.css Response Headers ---"
curl -sI https://goodai.co.nz/style.css || true

echo "--- Versioned style.css?v=20261014-01 Response Headers ---"
curl -sI "https://goodai.co.nz/style.css?v=20261014-01" || true

echo "--- Line count of style.css from public URL ---"
curl -s https://goodai.co.nz/style.css | wc -l || true
