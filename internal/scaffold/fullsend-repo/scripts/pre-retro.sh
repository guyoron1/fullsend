#!/usr/bin/env bash
# pre-retro.sh — Validate inputs for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue), checks GH_TOKEN validity, and logs the
# trigger context.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#
# Optional env vars:
#   GH_TOKEN        — GitHub token for API access
#   RETRO_COMMENT   — The /retro comment text (empty for automatic triggers)

set -euo pipefail

: "${ORIGINATING_URL:?ORIGINATING_URL is required}"

# Accept both issue and PR URLs.
if [[ ! "${ORIGINATING_URL}" =~ ^https://github\.com/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+/(issues|pull)/[0-9]+$ ]]; then
  echo "ERROR: ORIGINATING_URL does not match expected pattern"
  exit 1
fi

# ---------------------------------------------------------------------------
# GH_TOKEN validation — fail fast if token is set but invalid
# ---------------------------------------------------------------------------
if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "::warning::GH_TOKEN is not set — retro agent may not be able to access GitHub API"
else
  if ! gh auth status &>/dev/null; then
    echo "::error::GH_TOKEN is set but invalid — aborting to avoid wasting inference tokens"
    exit 1
  fi
  echo "GH_TOKEN validation passed."
fi

echo "::notice::Retro target: ${ORIGINATING_URL}"

if [[ -n "${RETRO_COMMENT:-}" ]]; then
  echo "Retro triggered on-demand with comment."
else
  echo "Retro triggered automatically (PR close)."
fi

echo "Pre-retro validation complete."
