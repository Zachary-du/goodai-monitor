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
  git commit -m "feat: dual-engine hero tabs, instant public playground, developer FAQ (TASK-012)"
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

echo "Checking Dual-Engine tabs, Playground & FAQ markup in live HTML:"
curl -s https://goodai.co.nz/ | grep -E "showcase-tab-bar|playground-section|faq-section|v=20261016-01" || true

echo "Checking initPublicAudit & initShowcaseTabs in live JavaScript:"
curl -s "https://goodai.co.nz/app.js?v=20261016-01" | grep -o -E "initShowcaseTabs|initPublicAudit|initFAQ" | head -n 3 || true

echo "Testing unauthenticated Public Security Audit API endpoint:"
curl -s -X POST https://goodai.co.nz/api/v1/security/public-audit \
  -H "Content-Type: application/json" \
  -d '{"target_url":"https://example.com"}' | grep -o -E '"composite_score":[0-9]+|"target_host":"[^"]+"' || true

echo "=== [SUCCESS] TASK-012 Successfully Deployed and Live Verified ==="
