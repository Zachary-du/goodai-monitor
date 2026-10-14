#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Checking git status and pushing updates to main ==="
git add web/index.html web/logo.svg web/embed.go web/style.css scripts/run.sh
git commit -m "feat(branding): update logo cache-busting version to 20261015-02" || true
git push origin main

echo "=== 2. Waiting 55s for GitHub Actions remote CI to finish compilation and container publish ==="
sleep 55

echo "=== 3. Pulling latest container image and recreating container on Melbourne VPS ==="
ssh -i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no ubuntu@158.179.29.27 << 'EOF'
set -e
cd /opt/goodai-monitor
echo "Pulling latest container image from GHCR..."
docker compose pull
echo "Recreating goodai-monitor container with official Option 3 brand logo..."
docker compose up -d
echo "Waiting 5 seconds for healthcheck..."
sleep 5
docker ps | grep goodai-monitor
docker logs --tail 20 goodai-monitor
EOF

echo "=== 4. Public Verification against https://goodai.co.nz ==="
echo "--- Root / Response Headers ---"
curl -sI https://goodai.co.nz/

echo "--- logo.svg Response Headers ---"
curl -sI "https://goodai.co.nz/logo.svg?v=20261015-02"

echo "--- Byte size of logo.svg ---"
curl -s "https://goodai.co.nz/logo.svg?v=20261015-02" | wc -c

echo "=== Brand Logo Deployment Complete ==="
