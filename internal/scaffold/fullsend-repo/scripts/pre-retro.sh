#!/usr/bin/env bash
# pre-retro.sh — Validate inputs and prefetch PR context for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue), prefetches PR data from GitHub, and
# writes it to a JSON file for the sandbox to consume.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#
# Optional env vars:
#   RETRO_COMMENT   — The /retro comment text (empty for automatic triggers)
#   GH_TOKEN        — GitHub token for API calls (prefetch skipped if unset)
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

# ---------------------------------------------------------------------------
# Prefetch PR context — runs on the host where the token and network are
# reliable, so the sandbox agent has data even if GH_TOKEN is invalid
# inside the sandbox.
# ---------------------------------------------------------------------------

# Extract repo and number from ORIGINATING_URL.
URL_TYPE=$(echo "${ORIGINATING_URL}" \
  | grep -oP '(?<=/)(?:issues|pull)(?=/[0-9]+$)')
URL_REPO=$(echo "${ORIGINATING_URL}" \
  | grep -oP '(?<=github\.com/)[^/]+/[^/]+')
URL_NUMBER=$(basename "${ORIGINATING_URL}")

PR_CONTEXT_FILE="${RUNNER_TEMP:-/tmp}/pr-context.json"

# Only prefetch for PR URLs — issue URLs have no PR-specific data.
if [[ "${URL_TYPE}" != "pull" ]]; then
  echo "Originating URL is an issue, not a PR — skipping prefetch."
  exit 0
fi

if [[ -z "${GH_TOKEN:-}" ]]; then
  echo "::warning::GH_TOKEN not set — skipping PR context prefetch"
  exit 0
fi

echo "Prefetching PR context for ${URL_REPO}#${URL_NUMBER}..."

prefetch_failed=false

# 1. PR metadata via gh pr view.
pr_metadata=""
if pr_metadata=$(gh pr view "${URL_NUMBER}" \
  --repo "${URL_REPO}" \
  --json number,title,body,state,author,labels,mergedBy,baseRefName,headRefName,additions,deletions,changedFiles,commits,createdAt,closedAt,mergedAt,mergeCommit,isDraft,reviewDecision \
  2>&1); then
  echo "Fetched PR metadata."
else
  echo "::warning::Failed to fetch PR metadata: ${pr_metadata}"
  prefetch_failed=true
fi

# 2. PR comments (issue comments endpoint covers both).
pr_comments=""
if ! ${prefetch_failed}; then
  if pr_comments=$(gh api "repos/${URL_REPO}/issues/${URL_NUMBER}/comments" \
    --paginate 2>&1); then
    echo "Fetched PR comments."
  else
    echo "::warning::Failed to fetch PR comments: ${pr_comments}"
    pr_comments="[]"
  fi
fi

# 3. PR reviews.
pr_reviews=""
if ! ${prefetch_failed}; then
  if pr_reviews=$(gh api "repos/${URL_REPO}/pulls/${URL_NUMBER}/reviews" \
    --paginate 2>&1); then
    echo "Fetched PR reviews."
  else
    echo "::warning::Failed to fetch PR reviews: ${pr_reviews}"
    pr_reviews="[]"
  fi
fi

# 4. Recent workflow runs for the default branch dispatch workflow.
#    Fetches the 10 most recent runs to give the retro agent CI context.
workflow_runs=""
if ! ${prefetch_failed}; then
  if workflow_runs=$(gh api "repos/${URL_REPO}/actions/runs" \
    --method GET \
    -f per_page=10 \
    --jq '{workflow_runs: [.workflow_runs[] | {id, name, status, conclusion, html_url, created_at, updated_at, head_sha, head_branch, event}]}' \
    2>&1); then
    echo "Fetched recent workflow runs."
  else
    echo "::warning::Failed to fetch workflow runs: ${workflow_runs}"
    workflow_runs='{"workflow_runs":[]}'
  fi
fi

# 5. Assemble and write the context file.
if ${prefetch_failed}; then
  echo "::warning::PR metadata fetch failed — writing empty context file"
  printf '{"prefetch_status":"failed","error":"PR metadata fetch failed"}\n' \
    > "${PR_CONTEXT_FILE}"
else
  jq -n \
    --arg status "ok" \
    --argjson metadata "${pr_metadata}" \
    --argjson comments "${pr_comments}" \
    --argjson reviews "${pr_reviews}" \
    --argjson workflow_runs "${workflow_runs}" \
    '{
      prefetch_status: $status,
      pr: $metadata,
      comments: $comments,
      reviews: $reviews,
      workflow_runs: $workflow_runs.workflow_runs
    }' > "${PR_CONTEXT_FILE}"
  echo "PR context written to ${PR_CONTEXT_FILE}"
fi

# Output the file path for workflow integration.
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  echo "pr_context_file=${PR_CONTEXT_FILE}" >> "${GITHUB_OUTPUT}"
fi

echo "Pre-retro prefetch complete."
