#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Checking git status and pushing documentation / config updates to main ==="
git add .env.example docs/ai-team/tasks/README.md docs/ai-team/tasks/TASK-009-google-oauth-sso.md scripts/run.sh
git commit -m "docs: register TASK-009 Google OAuth SSO and update .env.example" || true
git push origin main

echo "=== Sync complete ==="
