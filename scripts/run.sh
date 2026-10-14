#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Committing and pushing cache-busting version bump on main ==="
git add web/index.html scripts/run.sh
git commit -m "fix(cache): bump static assets version query parameter to v=20261014-08" || true
git push origin main

echo "=== 2. Waiting 50s for GitHub Actions remote CI to finish compilation ==="
sleep 50

echo "=== 3. Pulling latest container image and recreating container on Melbourne VPS ==="
ssh -i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no ubuntu@158.179.29.27 << 'EOF'
set -e
cd /opt/goodai-monitor
echo "Pulling latest container image from GHCR..."
docker compose pull
echo "Recreating goodai-monitor container..."
docker compose up -d
echo "Waiting 5 seconds for healthcheck..."
sleep 5
docker ps | grep goodai-monitor
docker logs --tail 20 goodai-monitor
EOF

echo "=== 4. Public Verification against https://goodai.co.nz ==="
echo "--- Root / Response Headers ---"
curl -sI https://goodai.co.nz/

echo "--- Versioned style.css?v=20261014-08 Response Headers ---"
curl -sI "https://goodai.co.nz/style.css?v=20261014-08"

echo "--- Versioned app.js?v=20261014-08 Response Headers ---"
curl -sI "https://goodai.co.nz/app.js?v=20261014-08"

echo "--- Byte size of Versioned app.js?v=20261014-08 ---"
curl -s "https://goodai.co.nz/app.js?v=20261014-08" | wc -c

echo "=== Deployment and Verification Complete ==="
