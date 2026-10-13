#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== Packaging and pushing TASK-006 ==="
git status
git add -A
git commit -m "feat(deploy): implement multi-stage Dockerfile, docker-compose, and GHCR pipeline (TASK-006)" || echo "No changes to commit"
git push -u origin task/TASK-006-containerization-and-deployment
echo "=== Pushed branch task/TASK-006-containerization-and-deployment ==="
