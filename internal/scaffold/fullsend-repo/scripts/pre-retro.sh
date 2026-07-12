#!/usr/bin/env bash
# pre-retro.sh — Validate inputs and prefetch PR context for the retro agent.
#
# Runs on the host via the harness pre_script mechanism. Validates the
# originating URL (PR or issue), then prefetches PR/issue data so the
# retro agent has context available even if GH_TOKEN is invalid inside
# the sandbox.
#
# Required env vars:
#   ORIGINATING_URL — HTML URL of the PR or issue that triggered retro
#
# Optional env vars:
#   RETRO_COMMENT   — The /retro comment text (empty for automatic triggers)
#   GH_TOKEN        — GitHub token for API calls (used for prefetch)
#   REPO_FULL_NAME  — Source repository (owner/repo)

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
# Prefetch PR/issue context
# ---------------------------------------------------------------------------
# Fetch data on the host where the token and network are reliable. This
# makes the retro agent resilient to in-sandbox token failures and reduces
# inference-time API calls. The prefetch is non-blocking: failures emit
# warnings but do not abort the script.

prefetch_context() {
  if [[ -z "${GH_TOKEN:-}" ]]; then
    echo "::warning::GH_TOKEN not set — skipping context prefetch"
    return 1
  fi

  local url="${ORIGINATING_URL}"
  local repo number url_type

  # Parse repo and number from ORIGINATING_URL.
  repo=$(echo "${url}" | sed -E 's#https://github\.com/##; s#/(issues|pull)/.*##')
  number=$(basename "${url}")
  if echo "${url}" | grep -q '/pull/'; then
    url_type="pull"
  else
    url_type="issue"
  fi

  local output_dir="${GITHUB_WORKSPACE:-.}"
  local output_file="${output_dir}/pr-context.json"

  if [[ "${url_type}" == "pull" ]]; then
    echo "Prefetching PR context for ${repo}#${number}..."

    # 1. PR metadata via gh pr view.
    local pr_meta
    pr_meta=$(gh pr view "${number}" --repo "${repo}" \
      --json title,body,state,author,labels,baseRefName,headRefName,headRefOid,additions,deletions,changedFiles,commits,createdAt,closedAt,mergedAt,mergedBy,number,url \
      2>/dev/null) || { echo "::warning::Failed to fetch PR metadata"; return 1; }

    # 2. PR comments (issue comments endpoint includes PR comments).
    local pr_comments
    pr_comments=$(gh api "repos/${repo}/issues/${number}/comments" \
      --paginate 2>/dev/null) || pr_comments="[]"

    # 3. PR reviews.
    local pr_reviews
    pr_reviews=$(gh api "repos/${repo}/pulls/${number}/reviews" \
      --paginate 2>/dev/null) || pr_reviews="[]"

    # 4. Recent workflow runs for the source repo, filtered to the PR
    #    lifecycle window. Uses the PR createdAt as the lower bound.
    local pr_created_at workflow_runs
    pr_created_at=$(echo "${pr_meta}" | jq -r '.createdAt // empty')
    if [[ -n "${pr_created_at}" ]]; then
      workflow_runs=$(gh api "repos/${repo}/actions/runs?per_page=20&created=>=${pr_created_at}" \
        --jq '.workflow_runs | map({id, name, status, conclusion, created_at, html_url, event})' \
        2>/dev/null) || workflow_runs="[]"
    else
      workflow_runs="[]"
    fi

    # Assemble the context JSON.
    jq -n \
      --argjson metadata "${pr_meta}" \
      --argjson comments "${pr_comments}" \
      --argjson reviews "${pr_reviews}" \
      --argjson workflow_runs "${workflow_runs}" \
      --arg source_url "${url}" \
      --arg source_repo "${repo}" \
      --arg source_number "${number}" \
      --arg source_type "${url_type}" \
      '{
        source_url: $source_url,
        source_repo: $source_repo,
        source_number: ($source_number | tonumber),
        source_type: $source_type,
        pr: $metadata,
        comments: $comments,
        reviews: $reviews,
        workflow_runs: $workflow_runs
      }' > "${output_file}"

  else
    echo "Prefetching issue context for ${repo}#${number}..."

    # Issue metadata.
    local issue_meta
    issue_meta=$(gh api "repos/${repo}/issues/${number}" \
      2>/dev/null) || { echo "::warning::Failed to fetch issue metadata"; return 1; }

    # Issue comments.
    local issue_comments
    issue_comments=$(gh api "repos/${repo}/issues/${number}/comments" \
      --paginate 2>/dev/null) || issue_comments="[]"

    # Assemble the context JSON.
    jq -n \
      --argjson metadata "${issue_meta}" \
      --argjson comments "${issue_comments}" \
      --arg source_url "${url}" \
      --arg source_repo "${repo}" \
      --arg source_number "${number}" \
      --arg source_type "${url_type}" \
      '{
        source_url: $source_url,
        source_repo: $source_repo,
        source_number: ($source_number | tonumber),
        source_type: $source_type,
        issue: $metadata,
        comments: $comments
      }' > "${output_file}"
  fi

  local byte_count
  byte_count=$(wc -c < "${output_file}")
  echo "Prefetched context written to ${output_file} (${byte_count} bytes)"

  # Sanity check: reject files over 5 MB to avoid bloating the sandbox.
  local max_bytes=5242880
  if [[ "${byte_count}" -gt "${max_bytes}" ]]; then
    echo "::warning::Prefetched context exceeds 5 MB (${byte_count} bytes) — discarding"
    rm -f "${output_file}"
    return 1
  fi
}

# Run prefetch; failures are non-blocking.
if prefetch_context; then
  echo "PR context prefetch complete."
else
  echo "::warning::PR context prefetch failed (non-blocking). The agent will attempt live API calls."
fi
