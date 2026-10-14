#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

BRANCH="task/TASK-008-security-audit-and-ai-defense-report"

echo "=== 1. Pushing branch $BRANCH to origin ==="
git push -u origin "$BRANCH"

echo "=== 2. Checking GitHub Actions Runs for $BRANCH ==="
sleep 3
curl -s "https://api.github.com/repos/Zachary-du/goodai-monitor/actions/runs?branch=${BRANCH}&per_page=1" | grep -E '"name"|"status"|"conclusion"|"head_branch"|"head_sha"' | head -n 10 || true

echo "=== Branch pushed and CI workflow triggered ==="
