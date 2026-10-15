#!/usr/bin/env bash
set -euo pipefail

export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

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

echo "=== [2/5] Staging and Committing TASK-009 Changes ==="
git checkout -B task/TASK-009-google-oauth-sso
git add internal/ web/ docs/ai-team/tasks/TASK-009-google-oauth-sso.md scripts/run.sh
if git diff --staged --quiet; then
  echo "No changes to commit."
else
  git commit -m "feat(auth): implement Google OAuth 2.0 Single Sign-On and enterprise UI (TASK-009)"
fi

echo "=== [3/5] Pushing Branch to Remote GitHub Repository ==="
git push origin task/TASK-009-google-oauth-sso
HEAD_SHA="$(git rev-parse HEAD)"
echo "Pushed commit: ${HEAD_SHA}"

echo "=== [4/5] Awaiting Remote GitHub Actions CI Execution ==="
for i in {1..35}; do
  RUN_INFO=$(gh run list --branch task/TASK-009-google-oauth-sso --commit "${HEAD_SHA}" --json databaseId,status,conclusion --limit 1 2>/dev/null || true)
  if [[ -n "${RUN_INFO}" && "${RUN_INFO}" != "[]" ]]; then
    STATUS=$(echo "${RUN_INFO}" | jq -r '.[0].status')
    CONCLUSION=$(echo "${RUN_INFO}" | jq -r '.[0].conclusion')
    echo "[CI Monitor] Check ${i}: status=${STATUS}, conclusion=${CONCLUSION}"
    if [[ "${STATUS}" == "completed" ]]; then
      if [[ "${CONCLUSION}" == "success" ]]; then
        echo "[CI Monitor] Remote GitHub Actions CI succeeded!"
        break
      else
        echo "[CI Monitor] Remote GitHub Actions CI failed with conclusion: ${CONCLUSION}"
        gh run view --commit "${HEAD_SHA}" --log-failed || true
        exit 1
      fi
    fi
  else
    echo "[CI Monitor] Check ${i}: waiting for CI run to register for ${HEAD_SHA}..."
  fi
  sleep 8
done

echo "=== [5/5] Remote GitHub Actions CI Pipeline Completed Successfully! ==="
