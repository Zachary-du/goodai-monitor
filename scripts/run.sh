#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Committing scripts/run.sh on task/BUG-007-01-cache-and-visual-polish ==="
git add scripts/run.sh
git commit -m "chore(ops): update scripts/run.sh for VPS deployment" || true
git push origin task/BUG-007-01-cache-and-visual-polish

echo "=== 2. Merging task/BUG-007-01-cache-and-visual-polish into main ==="
git checkout main
git pull origin main
git merge --no-ff task/BUG-007-01-cache-and-visual-polish -m "Merge task/BUG-007-01-cache-and-visual-polish into main (Cache-Busting & Visual Polish)"
git push origin main

echo "=== 3. Inspecting running containers on Melbourne VPS ==="
ssh -i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no ubuntu@158.179.29.27 "docker ps"

echo "=== 4. Completed Git Merge and Push ==="
