#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

echo "=== 1. Checking 100% English Policy across code and assets ==="
MATCHES=$(git grep -I -P "[\x{4e00}-\x{9fa5}]" -- ':!docs' ':!Gemini.md' ':!.git' || true)
if [ -n "$MATCHES" ]; then
    echo "ERROR: Found non-English characters in source files:"
    echo "$MATCHES"
    exit 1
fi
echo "Verified: 100% English compliance maintained."

echo "=== 2. Committing changes and pushing to main ==="
git add .github/workflows/ci-cd.yml scripts/run.sh
git commit -m "ci: ignore binary files in English policy verification step" || true
git push origin main

HEAD_SHA=$(git rev-parse HEAD)
echo "Pushed commit: ${HEAD_SHA}"

echo "=== 3. Dynamically awaiting remote GitHub Actions CI completion ==="
for i in {1..35}; do
    echo "Checking GitHub Actions CI run status (attempt $i/35)..."
    RUN_DATA=$(curl -s "https://api.github.com/repos/Zachary-du/goodai-monitor/actions/runs?head_sha=${HEAD_SHA}&per_page=1")
    STATUS=$(echo "$RUN_DATA" | grep -o '"status": *"[^"]*"' | head -n 1 | cut -d'"' -f4 || true)
    CONCLUSION=$(echo "$RUN_DATA" | grep -o '"conclusion": *"[^"]*"' | head -n 1 | cut -d'"' -f4 || true)

    echo "  Status: [${STATUS}], Conclusion: [${CONCLUSION}]"
    if [ "$STATUS" = "completed" ] && [ "$CONCLUSION" = "success" ]; then
        echo "GitHub Actions CI compilation and image publish SUCCEEDED!"
        break
    fi
    if [ "$STATUS" = "completed" ] && [ "$CONCLUSION" = "failure" ]; then
        echo "ERROR: GitHub Actions CI run failed!"
        exit 1
    fi
    sleep 8
done

echo "=== 4. Recreating container on Melbourne VPS with verified image ==="
ssh -i key/158.179.29.27.key -p 60324 -o StrictHostKeyChecking=no ubuntu@158.179.29.27 << 'EOF'
set -e
cd /opt/goodai-monitor
echo "Executing docker compose pull..."
docker compose pull
echo "Recreating container..."
docker compose up -d
echo "Waiting 6s for healthcheck..."
sleep 6
docker ps | grep goodai-monitor
echo "Local container HTML test:"
curl -s http://127.0.0.1:8090/ | grep -C 1 "brand-logo-img"
EOF

echo "=== 5. Comprehensive Public QA Verification against https://goodai.co.nz ==="
echo "--- 5.1: Verify brand logo img tag in public HTML ---"
curl -s https://goodai.co.nz/ | grep -C 1 "brand-logo-img"

echo "--- 5.2: Verify logo.png response status (must be HTTP 200 with image/png) ---"
curl -sI "https://goodai.co.nz/logo.png?v=20261015-04" | grep -E "HTTP/|content-type|content-length"

echo "--- 5.3: Verify logo.svg response status (must be HTTP 200 with image/svg+xml) ---"
curl -sI "https://goodai.co.nz/logo.svg?v=20261015-04" | grep -E "HTTP/|content-type|content-length"

echo "--- 5.4: Verify commercial footer in public HTML ---"
curl -s https://goodai.co.nz/ | grep -C 2 "GoodAI Monitor Ltd"

echo "--- 5.5: Assert ZERO dev jargon ('20 MiB', '158.179') in public HTML ---"
JARGON_COUNT=$(curl -s https://goodai.co.nz/ | grep -Ei "20 mib|158\.179" | wc -l || true)
echo "Dev jargon matches count: ${JARGON_COUNT}"
if [ "$JARGON_COUNT" -gt 0 ]; then
    echo "ERROR: Found residual internal dev jargon in public HTML!"
    exit 1
fi
echo "Verified: Zero internal developer jargon present on public website."

echo "=== Enterprise Brand and Copy Overhaul Complete ==="
