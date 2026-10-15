#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Sync documentation updates to main ==="
git add docs/ai-team/tasks/BUG-008-03-commercial-copy-and-logo-fidelity-qa-report.md \
        docs/ai-team/tasks/README.md \
        scripts/run.sh
git commit -m "docs: register BUG-008-03 QA audit report and resolution" || true
git push origin main

echo "=== Documentation synced ==="
