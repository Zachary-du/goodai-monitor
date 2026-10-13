#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Committing fix for unused import ==="
git status
git add -A
git commit -m "fix(alert): remove unused errors import from manager.go" || echo "No changes to commit"
git push origin task/TASK-005-incident-lifecycle-and-email-alerts
echo "=== Pushed branch task/TASK-005-incident-lifecycle-and-email-alerts ==="
