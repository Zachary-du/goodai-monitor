#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

git add scripts/run.sh
if git diff --staged --quiet; then
  echo "Clean."
else
  git commit -m "chore: clean up unified ops runner script"
  git push origin main
fi
echo "Repository clean on main: $(git rev-parse --short HEAD)"
