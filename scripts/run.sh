#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Packaging and pushing TASK-006 updates ==="
git status
git add -A
git commit -m "fix(ci): optimize Dockerfile build flags and add docker tarball artifact (TASK-006)" || echo "No changes to commit"
git push origin task/TASK-006-containerization-and-deployment
echo "=== Pushed branch task/TASK-006-containerization-and-deployment ==="
