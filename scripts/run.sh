#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== GoodAI Monitor Ops Runner ==="
git status
git add -A
git commit -m "feat(incident): implement incident state machine and smtp email alerts (TASK-005)" || echo "No changes to commit"
git push -u origin task/TASK-005-incident-lifecycle-and-email-alerts
echo "=== Pushed branch task/TASK-005-incident-lifecycle-and-email-alerts ==="
