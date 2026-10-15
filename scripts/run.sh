#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Sync documentation updates to main ==="
git add docs/ai-team/tasks/BUG-008-02-brand-logo-rendering.md \
        docs/ai-team/tasks/BUG-008-02-brand-logo-rendering-qa-report.md \
        docs/ai-team/tasks/README.md \
        scripts/run.sh
git commit -m "docs: complete BUG-008-02 QA audit report and register resolution" || true
git push origin main

echo "=== Sync complete ==="
