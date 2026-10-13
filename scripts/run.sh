#!/usr/bin/env bash
# GoodAI Monitor Unified Ops Runner Script
# Packaging commands with the same script name avoids repeated authorization prompts.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "[Ops Runner] Working directory: ${PROJECT_ROOT}"
