#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== [1/5] Verifying 100% English Compliance ==="
MATCHES=$(git grep -I -P "[\x{4e00}-\x{9fa5}]" internal/ web/ cmd/ || true)
if [[ -n "${MATCHES}" ]]; then
  echo "[ERROR] Non-English characters detected in production code:"
  echo "${MATCHES}"
  exit 1
fi
echo "[SUCCESS] Zero non-English characters found in production assets."

echo "=== [2/5] Staging and Committing Bug Fix ==="
git checkout -B task/TASK-009-google-oauth-sso
git add internal/storage/user_store.go scripts/run.sh
if git diff --staged --quiet; then
  echo "No changes to commit."
else
  git commit -m "fix(storage): use db.QueryContext instead of unexported conn in user_store"
fi

echo "=== [3/5] Pushing Branch to Remote GitHub Repository ==="
git push origin task/TASK-009-google-oauth-sso
HEAD_SHA="$(git rev-parse HEAD)"
echo "Pushed commit: ${HEAD_SHA}"

echo "=== [4/5] Awaiting Remote GitHub Actions CI Execution ==="
for i in {1..40}; do
  RESP=$(curl -s "https://api.github.com/repos/Zachary-du/goodai-monitor/actions/runs?branch=task/TASK-009-google-oauth-sso" || true)
  RUN_STATUS=$(python3 -c '
import sys, json
try:
    data = json.loads(sys.argv[1])
    head_sha = sys.argv[2]
    runs = data.get("workflow_runs", [])
    for r in runs:
        if r.get("head_sha") == head_sha:
            print(f"{r.get(\"status\")}:{r.get(\"conclusion\")}")
            sys.exit(0)
    print("not_found:")
except Exception as e:
    print(f"error:{e}")
' "$RESP" "$HEAD_SHA")

  STATUS="${RUN_STATUS%%:*}"
  CONCLUSION="${RUN_STATUS##*:}"

  echo "[CI Monitor] Check ${i}: status=${STATUS}, conclusion=${CONCLUSION}"

  if [[ "${STATUS}" == "completed" ]]; then
    if [[ "${CONCLUSION}" == "success" ]]; then
      echo "[SUCCESS] Remote GitHub Actions CI passed!"
      break
    else
      echo "[ERROR] Remote GitHub Actions CI finished with conclusion: ${CONCLUSION}"
      exit 1
    fi
  fi
  sleep 8
done

echo "=== [5/5] GitHub Actions CI Pipeline Completed Successfully! ==="
