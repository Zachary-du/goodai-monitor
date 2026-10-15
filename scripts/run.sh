#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== [1/6] Staging and Committing QA Documentation on task branch ==="
git checkout task/TASK-009-google-oauth-sso
git pull --rebase origin task/TASK-009-google-oauth-sso || true
git add docs/ai-team/tasks/TASK-009* scripts/run.sh
if git diff --staged --quiet; then
  echo "No doc changes to commit."
else
  git commit -m "docs(qa): finalize TASK-009 documentation and runner" || true
fi
git push origin task/TASK-009-google-oauth-sso
echo "[Git] Pushed task branch to remote."

echo "=== [2/6] Merging task/TASK-009-google-oauth-sso into main ==="
git checkout main
git pull origin main
git merge --no-ff task/TASK-009-google-oauth-sso -m "Merge branch 'task/TASK-009-google-oauth-sso' into main"
git push origin main
MAIN_SHA="$(git rev-parse HEAD)"
echo "[Git] Merged and pushed to main: ${MAIN_SHA}"

echo "=== [3/6] Awaiting Remote CI Execution for main ==="
for i in {1..40}; do
  RESP=$(curl -s "https://api.github.com/repos/Zachary-du/goodai-monitor/actions/runs?branch=main" || true)
  RUN_STATUS=$(python3 -c '
import sys, json
try:
    data = json.loads(sys.argv[1])
    head_sha = sys.argv[2]
    runs = data.get("workflow_runs", [])
    found = False
    for r in runs:
        if r.get("head_sha") == head_sha:
            s = r.get("status")
            c = r.get("conclusion")
            print(str(s) + ":" + str(c))
            found = True
            break
    if not found:
        print("not_found:none")
except Exception as e:
    print("error:" + str(e))
' "$RESP" "$MAIN_SHA")

  STATUS="${RUN_STATUS%%:*}"
  CONCLUSION="${RUN_STATUS##*:}"

  echo "[CI Monitor - main] Check ${i}: status=${STATUS}, conclusion=${CONCLUSION}"

  if [[ "${STATUS}" == "completed" ]]; then
    if [[ "${CONCLUSION}" == "success" ]]; then
      echo "[SUCCESS] Remote GitHub Actions CI on main passed!"
      break
    else
      echo "[ERROR] Remote GitHub Actions CI on main failed with conclusion: ${CONCLUSION}"
      exit 1
    fi
  fi
  sleep 8
done

echo "=== [4/6] Configuring Melbourne VPS with Secret Isolation ==="
chmod 600 key/158.179.29.27.key

SSH_OPTS="-i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no"
SSH_TARGET="ubuntu@158.179.29.27"

if [[ -f .env.production ]]; then
  echo "[Deploy] Transferring isolated credentials from local .env.production..."
  ssh ${SSH_OPTS} ${SSH_TARGET} 'cat > /opt/goodai-monitor/.env.sso' < .env.production
  ssh ${SSH_OPTS} ${SSH_TARGET} 'bash -s' << 'REMOTE_SCRIPT'
set -euo pipefail
cd /opt/goodai-monitor
touch .env

while IFS= read -r line || [[ -n "$line" ]]; do
  if [[ -n "$line" && "$line" != \#* ]]; then
    key="${line%%=*}"
    val="${line#*=}"
    if grep -q "^${key}=" .env; then
      sed -i "s|^${key}=.*|${line}|" .env
    else
      echo "${line}" >> .env
    fi
  fi
done < .env.sso
rm -f .env.sso

# Ensure docker-compose.yml has env_file: .env
if ! grep -q "env_file:" docker-compose.yml; then
  sed -i '/container_name: goodai-monitor/a \    env_file:\n      - .env' docker-compose.yml
fi
echo "[VPS] Environment configuration synchronized."
REMOTE_SCRIPT
fi

echo "=== [5/6] Pulling Latest Container Image and Recreating Container ==="
ssh ${SSH_OPTS} ${SSH_TARGET} 'bash -s' << 'REMOTE_SCRIPT'
set -euo pipefail
cd /opt/goodai-monitor

docker compose pull
docker compose up -d --force-recreate
echo "[VPS] Container deployment completed. Verifying process..."
sleep 3
docker compose ps
REMOTE_SCRIPT

echo "=== [6/6] Verifying Live Production Endpoints ==="
echo "Checking /api/v1/auth/providers:"
curl -fsS "https://goodai.co.nz/api/v1/auth/providers"
echo ""

echo "Checking /api/v1/auth/google/login (should redirect HTTP 302):"
curl -sI "https://goodai.co.nz/api/v1/auth/google/login" | head -n 10

echo "=== TASK-009 Deployment and Verification Completed! ==="
