# Fullsend Test Plan

## **Retro Post-Script Non-Fatal 403/401 Error Handling - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2305](https://github.com/fullsend-ai/fullsend/issues/2305) — Retro post-script should treat 403 comment-posting errors as non-fatal
- **Feature Tracking:** [GH-2305](https://github.com/fullsend-ai/fullsend/issues/2305)
- **Epic Tracking:** N/A — standalone bug fix
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QE test plan conventions apply. "Non-fatal" means the script exits 0 and logs a warning rather than exiting non-zero.

### Feature Overview

The retro agent post-script (`post-retro.sh`) currently treats all comment-posting failures as fatal, causing the entire GitHub Actions workflow run to fail even when the retro agent succeeded and proposal issues were filed. This change wraps the `gh api` comment-posting call in error handling that treats HTTP 401 and 403 responses as non-fatal warnings (logging via `::warning::`) while preserving fatal behavior for other HTTP errors (e.g., 500, 422). A new test file (`post-retro-test.sh`) provides 8 test cases covering happy paths, non-fatal permission errors, and fatal server errors using a mock `gh` command.

---

### Section I — Motivation and Requirements Review

#### 1. Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-2305 describes a clear defect: two retro workflow runs (26291574658, 26915715367) on `konflux-ci/caching#816` were marked as failed despite the retro agent succeeding (exit code 0, schema validation passed, proposal issue filed).
  - Root cause identified: `post-retro.sh` receives HTTP 403 when the GitHub App token lacks `issues:write` permission on the source repo, and `set -euo pipefail` causes the script to exit non-zero.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a platform operator, I need retro workflow runs to report success when the agent completes and proposals are filed, even if the summary comment cannot be posted due to permission restrictions, so that failure rates accurately reflect real agent failures.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - The behavior change is binary: 401/403 responses from the comment-posting `gh api` call should produce exit 0 + warning instead of exit 1. Other HTTP errors remain fatal. This is directly testable with mock `gh` commands.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Validation criteria from the issue: "The next retro run on a repo where the app lacks `issues:write` should complete with conclusion `success` (not `failure`). The logs should show a warning about the skipped comment. Any proposal issues should still be created."

- [ ] **Confirmed coverage for NFRs.**
  - No performance, scale, or security NFRs apply. The change is purely error-handling logic in a shell script that runs once per workflow execution.

#### 2. Known Limitations

- The fix only covers the comment-posting step in `post-retro.sh`. Other post-scripts (`post-triage.sh`, `post-code.sh`, `post-review.sh`, `post-prioritize.sh`) have similar comment-posting patterns that may also encounter 403 errors but are not addressed by this change.
- The `post-retro-test.sh` test file is not registered in the `executableFiles` map in `internal/scaffold/scaffold.go`. If a `TestFileModeMatchesFilesystem` test validates that all executable scripts in the scaffold are tracked, this omission could cause a test failure. (Note: other test scripts like `post-code-test.sh` and `post-review-test.sh` are also absent from the map, so this may be intentional.)
- The 401/403 detection relies on grep-matching `"HTTP (401|403)"` in the stderr output of the `gh` CLI. If `gh` changes its error message format, the detection could silently break, causing the script to fall through to the fatal error path.

#### 3. Technology and Design Review

- [ ] **Reviewed Developer Handoff.**
  - PR #2306 modifies `post-retro.sh` (16 additions, 2 deletions) and adds `post-retro-test.sh` (266 lines). The change captures the exit code and stderr from the `gh api` call, then branches on whether the error message contains "HTTP 401" or "HTTP 403".

- [ ] **Identified Technology Challenges.**
  - The fix uses bash string matching (`grep -qE`) on CLI stderr output, which is inherently fragile. However, the `gh` CLI error format has been stable and this approach is consistent with the existing error-handling style in other post-scripts.

- [ ] **Assessed Test Environment Needs.**
  - Tests use a mock `gh` binary injected via `PATH` override. No real GitHub API calls or cluster access required. Tests are self-contained bash scripts.

- [ ] **Reviewed API Extensions.**
  - No API changes. The fix modifies error handling around an existing `gh api` call to the GitHub Issues API (`repos/{owner}/{repo}/issues/{number}/comments`).

- [ ] **Evaluated Topology or Infrastructure Needs.**
  - No infrastructure changes. The script runs on GitHub Actions runners in the existing workflow configuration.

---

### Section II — Test Planning

#### 1. Scope of Testing

This test plan covers the error handling behavior of the `post-retro.sh` script when the `gh api` call to post a summary comment returns various HTTP error codes. The change is scoped to a single shell script in the scaffold layer.

**Testing Goals:**

- **P0:** Verify that 401/403 errors on comment posting result in exit 0 with a `::warning::` log message
- **P0:** Verify that other HTTP errors (500, 422) on comment posting remain fatal (exit non-zero)
- **P1:** Verify that proposal issues are still created regardless of comment-posting outcome
- **P1:** Verify happy-path behavior is preserved (comment posted successfully, exit 0)
- **P2:** Verify warning message contains repo and PR number for debugging

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub API permission model testing** — The 403 behavior is a GitHub platform concern; we only test the script's response to it.
- [ ] **Other post-scripts (post-triage.sh, post-code.sh, etc.)** — Out of scope per GH-2305; tracked separately.
- [ ] **Real GitHub API integration testing** — Tests use mock `gh` commands; end-to-end GitHub integration is covered by production retro runs.
- [ ] **Retro agent logic** — The agent itself is not modified; only the post-script error handling changes.

#### 2. Test Strategy

**Functional:**

- [ ] **Functional Testing**
  - Applicable: Y
  - Test the `post-retro.sh` script with mock inputs simulating 401, 403, 500, and 422 HTTP responses from `gh api`, as well as successful comment posting.

- [ ] **Automation Testing**
  - Applicable: Y
  - The `post-retro-test.sh` script provides automated test coverage with 8 test cases using a mock `gh` binary. Tests run in CI via pre-commit or directly via bash.

- [ ] **Regression Testing**
  - Applicable: Y
  - Happy-path tests verify that existing behavior (successful comment posting, proposal filing) is preserved. Fatal-error tests verify that 500/422 errors still cause script failure.

- [ ] **Upgrade Testing**
  - Applicable: N — The script is embedded via scaffold; no version upgrade path applies.

**Non-Functional:**

- [ ] **Performance Testing**
  - Applicable: N — Single script execution, no performance sensitivity.

- [ ] **Scale Testing**
  - Applicable: N — Script processes one retro result per invocation.

- [ ] **Security Testing**
  - Applicable: N — No security-sensitive changes; GH_TOKEN masking already exists.

- [ ] **Usability Testing**
  - Applicable: N — No user-facing interface changes.

- [ ] **Monitoring**
  - Applicable: Y
  - The `::warning::` annotation surfaces in GitHub Actions UI, providing visibility into skipped comments without marking the run as failed.

**Integration & Compatibility:**

- [ ] **Compatibility Testing**
  - Applicable: N — No version compatibility concerns for a shell script change.

- [ ] **Dependencies**
  - Applicable: Y
  - Depends on `gh` CLI error message format containing "HTTP 401" or "HTTP 403". Verified against current `gh` CLI behavior.

- [ ] **Cross Integrations**
  - Applicable: N — No cross-component integrations affected.

**Infrastructure:**

- [ ] **Cloud Testing**
  - Applicable: N — Runs on standard GitHub Actions runners.

#### 3. Test Environment

- **Cluster Topology:** N/A — no cluster required; tests run locally or in CI
- **Platform Version:** GitHub Actions runner (ubuntu-latest)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Temporary filesystem (mktemp)
- **Network:** No network access required (mock `gh` binary)
- **Operators:** N/A
- **Platform:** Bash 4+, jq, standard POSIX utilities
- **Special Configs:** Mock `gh` binary injected via `PATH` override; `GH_MOCK_COMMENT_FAIL` environment variable controls mock behavior

##### 3.1. Testing Tools & Frameworks

No new or special tools required. Tests use standard bash scripting with a mock `gh` binary.

#### 4. Entry Criteria

- [ ] PR #2306 is merged to main
- [ ] `post-retro-test.sh` passes all 8 test cases locally
- [ ] Pre-commit hooks pass (shellcheck validation)
- [ ] `post-retro.sh` is syntactically valid bash (`bash -n`)

#### 5. Risks

- [ ] **Timeline**
  - Risk: None — change is small and self-contained
  - Mitigation: N/A
  - Status: Low

- [ ] **Coverage**
  - Risk: Other post-scripts (`post-triage.sh`, `post-code.sh`, `post-review.sh`, `post-prioritize.sh`) may have the same 403 fatal-failure pattern but are not covered by this fix
  - Mitigation: File follow-up issues to apply the same pattern to other post-scripts
  - Status: Acknowledged — distinct from #1296 (transient 5xx retry) and #2058 (agent API key auth)

- [ ] **Environment**
  - Risk: Tests rely on mock `gh` binary; real GitHub API behavior could differ in edge cases
  - Mitigation: Validate on 3+ production retro runs across repos with restricted permissions (per validation criteria in GH-2305)
  - Status: Medium — production validation planned

- [ ] **Untestable**
  - Risk: The `grep -qE "HTTP (401|403)"` pattern depends on `gh` CLI error message format stability
  - Mitigation: Pin `gh` CLI version in CI; monitor for format changes in `gh` releases
  - Status: Low — format has been stable across `gh` versions

- [ ] **Resources**
  - Risk: None — no additional infrastructure needed
  - Mitigation: N/A
  - Status: Low

- [ ] **Dependencies**
  - Risk: `post-retro-test.sh` may not be in the `executableFiles` map in `scaffold.go`, potentially failing `TestFileModeMatchesFilesystem`
  - Mitigation: Verify test passes; add to `executableFiles` map if needed
  - Status: Needs verification

- [ ] **Other**
  - Risk: None identified
  - Mitigation: N/A
  - Status: Low

---

### Section III — Requirements Mapping

#### 1. Requirements-to-Tests Mapping

- **GH-2305** — HTTP 401/403 comment-posting errors are treated as non-fatal warnings
  - Verify 403 error on comment posting exits 0 with warning | Functional | P0
  - Verify 401 error on comment posting exits 0 with warning | Functional | P0
  - Verify warning message contains repo and PR identifier | Functional | P1
  - Verify 500 error on comment posting remains fatal | Functional | P0
  - Verify 422 error on comment posting remains fatal | Functional | P0

- **GH-2305** — Happy-path behavior preserved after error handling changes
  - Verify successful comment posting with one proposal | Functional | P0
  - Verify successful comment posting with no proposals | Functional | P1
  - Verify proposal issues created before comment posting | Functional | P1

- **GH-2305** — Non-fatal error handling across proposal states
  - Verify 403 with no proposals still exits 0 | Functional | P1
  - Verify 403 with multiple proposals still exits 0 | Functional | P2
  - Verify "Post-retro complete" message on successful run | Functional | P2

- **GH-2305** — Production validation of non-fatal 403 handling
  - Verify retro run succeeds on repo without issues:write permission | End-to-End | P0
  - Verify proposal issues created despite comment-posting 403 | End-to-End | P0
  - Verify GitHub Actions warning annotation visible in workflow logs | End-to-End | P1

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| Product Owner | | |
