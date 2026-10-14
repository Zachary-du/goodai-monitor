#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

BRANCH="task/TASK-008-security-audit-and-ai-defense-report"

echo "=== 1. Committing scripts/run.sh on ${BRANCH} ==="
git add scripts/run.sh
git commit -m "chore(ops): update scripts/run.sh for TASK-008 merge" || true
git push origin "${BRANCH}"

echo "=== 2. Merging ${BRANCH} into main ==="
git checkout main
git pull origin main
git merge --no-ff "${BRANCH}" -m "Merge task/TASK-008-security-audit-and-ai-defense-report into main (Web Security & AI Defense Audit)"
git push origin main

echo "=== 3. Successfully merged and pushed main to origin ==="
