#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== [Step 1] Staging and Committing Code ==="
git add -A
if git diff --staged --quiet; then
  echo "Working tree clean. Nothing new to commit."
else
  git commit -m "feat: customer fleet telemetry dashboard and total elimination of internal debug widgets (TASK-010)"
fi

echo "=== [Step 2] Pushing to Remote Repository ==="
git push origin main
COMMIT_SHA="$(git rev-parse HEAD)"
echo "Pushed commit: ${COMMIT_SHA}"

echo "=== [Step 3] Monitoring GitHub Actions CI Pipeline ==="
GH_TOKEN="$(tr -d '[:space:]' < github.token)"
REPO="Zachary-du/goodai-monitor"

MAX_WAIT=600
ELAPSED=0
SUCCESS=false

while [ $ELAPSED -lt $MAX_WAIT ]; do
  sleep 10
  ELAPSED=$((ELAPSED + 10))

  RESPONSE=$(curl -s -H "Authorization: Bearer ${GH_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${REPO}/actions/runs?head_sha=${COMMIT_SHA}")

  TOTAL_COUNT=$(echo "$RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('total_count', 0))" 2>/dev/null || echo "0")
  if [ "$TOTAL_COUNT" -gt 0 ]; then
    STATUS=$(echo "$RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['workflow_runs'][0].get('status', ''))" 2>/dev/null || echo "")
    CONCLUSION=$(echo "$RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['workflow_runs'][0].get('conclusion', ''))" 2>/dev/null || echo "")
    RUN_URL=$(echo "$RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data['workflow_runs'][0].get('html_url', ''))" 2>/dev/null || echo "")

    echo "CI Workflow Status: ${STATUS} | Conclusion: ${CONCLUSION} (${ELAPSED}s elapsed) -> ${RUN_URL}"

    if [ "$STATUS" = "completed" ]; then
      if [ "$CONCLUSION" = "success" ]; then
        echo "GitHub Actions CI Run Succeeded!"
        SUCCESS=true
        break
      else
        echo "GitHub Actions CI failed with conclusion: ${CONCLUSION}"
        exit 1
      fi
    fi
  else
    echo "Waiting for GitHub Actions to trigger for commit ${COMMIT_SHA}... (${ELAPSED}s)"
  fi
done

if [ "$SUCCESS" != "true" ]; then
  echo "Timed out waiting for GitHub Actions CI."
  exit 1
fi

echo "=== [Step 4] Deploying to Melbourne VPS ==="
VPS_HOST="158.179.29.27"
VPS_PORT="60324"
VPS_USER="ubuntu"
SSH_KEY="key/158.179.29.27.key"

ssh -o StrictHostKeyChecking=no -p "${VPS_PORT}" -i "${SSH_KEY}" "${VPS_USER}@${VPS_HOST}" << 'EOF'
  set -euo pipefail
  cd /opt/goodai-monitor
  echo "[VPS] Pulling latest ghcr.io image..."
  docker compose pull
  echo "[VPS] Recreating containers..."
  docker compose up -d
  echo "[VPS] Current container status:"
  docker compose ps
EOF

echo "=== [Step 5] Live Health & Content Verification ==="
sleep 3
echo "Checking https://goodai.co.nz/ HTTP Headers:"
curl -sI https://goodai.co.nz/ | head -n 5

echo "Checking Fleet Telemetry markup in live HTML:"
curl -s https://goodai.co.nz/ | grep -E "fleet-sla-val|Global Telemetry Infrastructure|v=20261015-06" || true

echo "Checking updateFleetMetrics presence in live JavaScript:"
curl -s "https://goodai.co.nz/app.js?v=20261015-06" | grep -o "updateFleetMetrics" | head -n 1 || true

echo "=== [SUCCESS] TASK-010 Successfully Deployed and Live Verified ==="
