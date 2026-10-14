#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Committing and pushing Option 3 Brand Logo to main ==="
git add web/logo.svg web/embed.go web/index.html web/style.css scripts/run.sh
git commit -m "feat(branding): implement Option 3 Pulse Waveform Monogram as brand logo and favicon" || true
git push origin main

echo "=== 2. Waiting 50s for GitHub Actions remote CI to finish compilation ==="
sleep 50

echo "=== 3. Pulling latest container image and recreating container on Melbourne VPS ==="
ssh -i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no ubuntu@158.179.29.27 << 'EOF'
set -e
cd /opt/goodai-monitor
echo "Pulling latest container image from GHCR..."
docker compose pull
echo "Recreating goodai-monitor container with official brand logo..."
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
curl -sI "https://goodai.co.nz/logo.svg?v=20261015-01"

echo "--- Byte size of logo.svg ---"
curl -s "https://goodai.co.nz/logo.svg?v=20261015-01" | wc -c

echo "=== Brand Logo Deployment Complete ==="
