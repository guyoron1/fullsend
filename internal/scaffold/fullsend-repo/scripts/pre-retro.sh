#!/usr/bin/env bash
# pre-retro.sh — Validate inputs and prefetch PR context for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue), logs the trigger context, and for PR
# triggers prefetches core PR data to make the retro agent resilient
# to in-sandbox token failures.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#
# Optional env vars:
#   GH_TOKEN        — GitHub token for API calls (prefetch skipped if unset)
#   RETRO_COMMENT   — The /retro comment text (empty for automatic triggers)
#   RUNNER_TEMP     — Temp directory for prefetched data (defaults to /tmp)

set -euo pipefail

: "${ORIGINATING_URL:?ORIGINATING_URL is required}"

# Accept both issue and PR URLs.
if [[ ! "${ORIGINATING_URL}" =~ ^https://github\.com/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+/(issues|pull)/[0-9]+$ ]]; then
  echo "ERROR: ORIGINATING_URL does not match expected pattern"
  exit 1
fi

echo "::notice::Retro target: ${ORIGINATING_URL}"

if [[ -n "${RETRO_COMMENT:-}" ]]; then
  echo "Retro triggered on-demand with comment."
else
  echo "Retro triggered automatically (PR close)."
fi

echo "Pre-retro validation complete."

# --- PR context prefetch ---
#
# For PR URLs, prefetch core data from the GitHub API on the host (where
# tokens and network access are reliable) so the retro agent has context
# even if the in-sandbox token is invalid. Each fetch degrades gracefully
# — partial data is better than none.

# Parse URL components (regex already validated above).
[[ "${ORIGINATING_URL}" =~ /([^/]+/[^/]+)/(issues|pull)/([0-9]+)$ ]]
PREFETCH_REPO="${BASH_REMATCH[1]}"
URL_KIND="${BASH_REMATCH[2]}"
PREFETCH_PR="${BASH_REMATCH[3]}"

# Only prefetch for PR URLs — issue retros do not need PR metadata.
if [[ "${URL_KIND}" != "pull" ]]; then
  echo "Originating URL is an issue, skipping PR context prefetch."
  exit 0
fi

# Token required for API calls.
if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "::warning::GH_TOKEN not set, skipping PR context prefetch."
  exit 0
fi

PREFETCH_DIR="${RUNNER_TEMP:-/tmp}"
CONTEXT_FILE="${PREFETCH_DIR}/pr-context.json"
PREFETCH_STATUS="full"

PREFETCH_TMP=$(mktemp -d)
trap 'rm -rf "${PREFETCH_TMP}"' EXIT

echo "Prefetching PR context: ${PREFETCH_REPO}#${PREFETCH_PR}"

# 1. PR metadata via gh pr view.
if gh pr view "${PREFETCH_PR}" --repo "${PREFETCH_REPO}" \
  --json title,body,state,author,labels,mergedBy,baseRefName,headRefName,additions,deletions,changedFiles,commits,createdAt,closedAt,mergedAt \
  > "${PREFETCH_TMP}/metadata.json" 2>/dev/null; then
  echo "Fetched PR metadata."
else
  echo "::warning::Failed to fetch PR metadata."
  echo '{}' > "${PREFETCH_TMP}/metadata.json"
  PREFETCH_STATUS="partial"
fi

# 2. PR comments (paginated, capped at 500 entries).
# --jq '.[]' with --paginate produces valid NDJSON across pages;
# jq -s reassembles into a single capped array.
if gh api "repos/${PREFETCH_REPO}/issues/${PREFETCH_PR}/comments?per_page=100" \
  --paginate --jq '.[]' 2>/dev/null \
  | jq -s '.[0:500]' > "${PREFETCH_TMP}/comments.json" 2>/dev/null; then
  COMMENT_COUNT=$(jq 'length' "${PREFETCH_TMP}/comments.json" 2>/dev/null || echo 0)
  echo "Fetched PR comments: ${COMMENT_COUNT} entries."
else
  echo "::warning::Failed to fetch PR comments."
  echo '[]' > "${PREFETCH_TMP}/comments.json"
  PREFETCH_STATUS="partial"
fi

# 3. PR reviews (paginated, capped at 500 entries).
if gh api "repos/${PREFETCH_REPO}/pulls/${PREFETCH_PR}/reviews?per_page=100" \
  --paginate --jq '.[]' 2>/dev/null \
  | jq -s '.[0:500]' > "${PREFETCH_TMP}/reviews.json" 2>/dev/null; then
  REVIEW_COUNT=$(jq 'length' "${PREFETCH_TMP}/reviews.json" 2>/dev/null || echo 0)
  echo "Fetched PR reviews: ${REVIEW_COUNT} entries."
else
  echo "::warning::Failed to fetch PR reviews."
  echo '[]' > "${PREFETCH_TMP}/reviews.json"
  PREFETCH_STATUS="partial"
fi

# 4. Recent workflow runs (10 most recent, no pagination needed).
if gh api "repos/${PREFETCH_REPO}/actions/runs?per_page=10" \
  --jq '.workflow_runs' > "${PREFETCH_TMP}/runs.json" 2>/dev/null; then
  RUN_COUNT=$(jq 'length' "${PREFETCH_TMP}/runs.json" 2>/dev/null || echo 0)
  echo "Fetched workflow runs: ${RUN_COUNT} entries."
else
  echo "::warning::Failed to fetch workflow runs."
  echo '[]' > "${PREFETCH_TMP}/runs.json"
  PREFETCH_STATUS="partial"
fi

# Assemble all data into a single JSON file.
if ! jq -n \
  --slurpfile metadata "${PREFETCH_TMP}/metadata.json" \
  --slurpfile comments "${PREFETCH_TMP}/comments.json" \
  --slurpfile reviews "${PREFETCH_TMP}/reviews.json" \
  --slurpfile runs "${PREFETCH_TMP}/runs.json" \
  --arg status "${PREFETCH_STATUS}" \
  --arg repo "${PREFETCH_REPO}" \
  --arg pr "${PREFETCH_PR}" \
  '{
    prefetch_status: $status,
    repo: $repo,
    pr_number: ($pr | tonumber),
    metadata: $metadata[0],
    comments: $comments[0],
    reviews: $reviews[0],
    workflow_runs: $runs[0]
  }' > "${CONTEXT_FILE}" 2>/dev/null; then
  echo "::warning::Failed to assemble pr-context.json, writing fallback."
  printf '{"prefetch_status":"failed","repo":"%s","pr_number":%s}\n' \
    "${PREFETCH_REPO}" "${PREFETCH_PR}" > "${CONTEXT_FILE}"
fi

# Size guard: cap at 5 MB to prevent oversized sandbox mounts.
FILE_SIZE=$(wc -c < "${CONTEXT_FILE}")
MAX_BYTES=$((5 * 1024 * 1024))
if [[ "${FILE_SIZE}" -gt "${MAX_BYTES}" ]]; then
  echo "::warning::pr-context.json exceeds 5 MB (${FILE_SIZE} bytes), writing metadata-only fallback."
  jq -n \
    --slurpfile metadata "${PREFETCH_TMP}/metadata.json" \
    --arg status "truncated" \
    --arg repo "${PREFETCH_REPO}" \
    --arg pr "${PREFETCH_PR}" \
    '{
      prefetch_status: $status,
      repo: $repo,
      pr_number: ($pr | tonumber),
      metadata: $metadata[0],
      comments: [],
      reviews: [],
      workflow_runs: []
    }' > "${CONTEXT_FILE}" 2>/dev/null || \
  printf '{"prefetch_status":"truncated","repo":"%s","pr_number":%s}\n' \
    "${PREFETCH_REPO}" "${PREFETCH_PR}" > "${CONTEXT_FILE}"
  FILE_SIZE=$(wc -c < "${CONTEXT_FILE}")
fi

echo "PR context written to ${CONTEXT_FILE} (${FILE_SIZE} bytes, status: ${PREFETCH_STATUS})."

# Export file path for harness host_files.
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "pr_context_file=${CONTEXT_FILE}" >> "${GITHUB_OUTPUT}"
fi
