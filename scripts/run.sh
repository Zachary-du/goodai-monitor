#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Checking GitHub Actions Runs via Public API ==="
curl -s "https://api.github.com/repos/Zachary-du/goodai-monitor/actions/runs?per_page=1" | grep -E '"name"|"status"|"conclusion"|"head_branch"|"head_sha"' | head -n 10

echo "=== 2. Pulling latest image and restarting goodai-monitor on Melbourne VPS ==="
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

echo "=== 3. Public Verification against https://goodai.co.nz ==="
echo "--- Root / Headers ---"
curl -sI https://goodai.co.nz/

echo "--- Versioned style.css?v=20261014-01 Headers ---"
curl -sI "https://goodai.co.nz/style.css?v=20261014-01"

echo "--- Line count of Versioned style.css?v=20261014-01 ---"
curl -s "https://goodai.co.nz/style.css?v=20261014-01" | wc -l

echo "--- Versioned app.js?v=20261014-01 Headers ---"
curl -sI "https://goodai.co.nz/app.js?v=20261014-01"

echo "=== Deployment and Verification Complete ==="
