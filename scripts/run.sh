#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Committing scripts/run.sh and merging to main ==="
git add scripts/run.sh
git commit -m "chore(scripts): finalize runner script" || echo "No changes to commit"
git push origin task/TASK-005-incident-lifecycle-and-email-alerts

git checkout main
git pull origin main
git merge --no-ff task/TASK-005-incident-lifecycle-and-email-alerts -m "Merge branch 'task/TASK-005-incident-lifecycle-and-email-alerts' into main (TASK-005 DONE)"
git push origin main

echo "=== Merge to main complete and pushed ==="
