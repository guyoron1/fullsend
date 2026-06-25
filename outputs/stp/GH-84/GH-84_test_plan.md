# Test Plan

## **fix(post-code): block workflow file changes and correct token docs - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-84](https://github.com/guyoron1/fullsend/issues/84)
- **Feature Tracking:** [GH-84](https://github.com/guyoron1/fullsend/issues/84)
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** "Post-code" refers to the script responsible for pushing agent commits and creating PRs after sandbox execution. "Workflow file block" refers to the defense-in-depth gate that rejects agent commits containing `.github/workflows/` modifications.

### Feature Overview

This change hardens the fullsend code agent's post-code pipeline against prompt injection attacks that could cause the agent to create malicious GitHub Actions workflow files. It adds an explicit script-level gate in `post-code.sh` (section 2c) that blocks any agent commit touching `.github/workflows/`, ensuring protection survives future token role changes or GitHub permission model shifts. Additionally, it corrects misleading comments in `code-agent.env` that incorrectly described the sandbox `GH_TOKEN` as read-only when it actually grants write permissions. Six new unit tests validate the workflow file detection logic.

---

### Section I - Motivation & Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [x] **Reviewed the relevant requirements.**
  - GH-84 describes three changes: workflow file blocking in post-code.sh, token documentation correction in code-agent.env, and 6 unit tests for workflow detection logic.
  - The issue references a team sync discussion on code agent security (prompt injection attack vector).

- [x] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - The value proposition is security hardening: preventing a prompt-injected agent from pushing malicious workflow files that execute arbitrary code on the org's runner with access to repo secrets.
  - User story: As a fullsend operator, I need defense-in-depth against workflow file injection so that even if the coder token gains `workflows:write` in the future, the script-level gate prevents exploitation.

- [x] **Confirmed requirements are **testable and unambiguous**.**
  - The workflow blocking logic is deterministic: any file matching `.github/workflows/*` in the changed files list triggers a hard block (exit 1).
  - Token documentation changes are informational and verifiable by code review.

- [x] **Ensured acceptance criteria are **defined clearly**.**
  - Acceptance criteria from the issue: all 58 post-code tests pass (including 6 new), shellcheck clean, security self-review approved, CI passes.

- [x] **Confirmed coverage for NFRs.**
  - Security: defense-in-depth against prompt injection attack vector.
  - Reliability: existing post-code pipeline behavior preserved (regression).

#### I.2 - Known Limitations

- The workflow file block operates on file paths only (pattern matching `.github/workflows/*`). It does not inspect file contents for malicious payloads in non-workflow files.
- The `code-agent.env` token documentation correction is informational only; it does not change any runtime behavior or token permissions.
- The defense-in-depth gate assumes the coder token's `workflows:write` omission remains the primary enforcement; the script-level block is a secondary layer.
- A TODO remains for minting a separate read-only sandbox token (noted in the issue but not implemented in this PR).

#### I.3 - Technology and Design Review

- [x] **Developer handoff completed.**
  - PR #84 modifies `internal/scaffold/fullsend-repo/scripts/post-code.sh` (new section 2c), `internal/scaffold/fullsend-repo/scripts/post-code-test.sh` (6 new tests), and `internal/scaffold/fullsend-repo/env/code-agent.env` (comment correction).

- [x] **Technology challenges identified.**
  - Shell script testing relies on function extraction and isolated execution (no git repo or network access needed for tests).
  - The scaffold embed system bundles post-code.sh into the Go binary via `embed.FS`. Changes to embedded files propagate automatically to all repos using the scaffold system.

- [x] **Test environment needs assessed.**
  - Unit tests for workflow detection run in bash without external dependencies.
  - Integration testing of the full post-code pipeline requires a GitHub Actions runner context (PUSH_TOKEN, REPO_FULL_NAME, etc.).

- [x] **API extensions reviewed.**
  - No API changes. The workflow block is internal to the post-code script.

- [x] **Topology and deployment considerations reviewed.**
  - The post-code.sh script runs on the GitHub Actions runner (not in the sandbox). Changes affect all repos using fullsend's scaffold system.

### Section II - Test Planning

#### II.1 - Scope of Testing

This test plan covers the security hardening changes to the post-code pipeline: the new workflow file blocking gate, the corrected token documentation, and regression of existing post-code functionality.

**Testing Goals:**

- **P0:** Verify workflow file detection blocks `.github/workflows/` changes with file path and blocking reason in output.
- **P0:** Verify existing post-code pipeline functionality is preserved (all 52 existing tests pass).
- **P1:** Verify edge cases in workflow detection (nested paths, non-workflow .github/ files, empty input).
- **P1:** Verify modified shell scripts pass shellcheck without new warnings.
- **P1:** Verify error output does not allow GitHub Actions command injection.
- **P2:** Verify error reporting comment includes correct workflow run URL.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub's server-side `workflows:write` enforcement.** Platform-level permission enforcement is owned by GitHub; the script-level block is a defense-in-depth addition.
- [ ] **Sandbox token minting changes.** The TODO for a separate read-only sandbox token is acknowledged but not implemented in this PR.
- [ ] **Prompt injection detection or prevention in the agent itself.** This PR addresses the post-code gate, not the agent's input handling.
- [ ] **Full end-to-end GitHub Actions workflow execution.** The post-code script is tested via extracted function unit tests; full pipeline testing requires CI infrastructure.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Core workflow file detection and blocking logic (security gate validation), post-code pipeline regression.
  - All 6 new workflow detection tests validate the defense-in-depth security gate as functional behavior.
  - All 52 existing post-code tests confirm regression coverage.
- [x] **Automation Testing** -- All tests are automated shell scripts executable via `bash post-code-test.sh`.
  - No manual test steps required.
- [x] **Regression Testing** -- Existing post-code behaviors preserved: title rewriting, PR body assembly, no-op detection, stale branch cleanup, push retry logic, error comment generation, artifact stripping, signed-off-by detection.
  - All existing test functions preserved and passing.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable; shell script execution is near-instantaneous.
- [ ] **Scale Testing** -- Not applicable; post-code processes a single PR at a time.
- [ ] **Security Testing** -- Not applicable; this change adds a functional security gate, not a security testing methodology. The workflow file blocking logic is validated under Functional Testing.
- [ ] **Usability Testing** -- Not applicable; no user-facing UI changes.
- [ ] **Monitoring** -- Not applicable; no new observability instrumentation.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable; bash script compatible with existing runner environments.
- [ ] **Upgrade Testing** -- Not applicable; scaffold embedding handles propagation.
- [x] **Dependencies** -- Updated post-code.sh is propagated to installed repos via the scaffold embedding system.
  - Existing scaffold tests validate that embedded file changes propagate correctly.
- [ ] **Cross Integrations** -- Not applicable; post-code.sh is self-contained.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable; tests run locally without cloud infrastructure.

#### II.3 - Test Environment

- **Cluster Topology:** N/A -- no cluster required for shell script unit tests
- **Platform Version:** GitHub Actions runner (Ubuntu latest)
- **CPU Virtualization:** N/A
- **Compute:** Standard GitHub Actions runner
- **Special Hardware:** None
- **Storage:** Local filesystem only
- **Network:** No network access required for unit tests
- **Operators:** N/A
- **Platform:** Linux (bash 4.0+, coreutils, grep, sed)
- **Special Configs:** None -- tests run in isolation without git repo or GitHub API

#### II.3.1 - Testing Tools & Frameworks

No new or special tools required. Tests use standard bash with built-in assertions.

#### II.4 - Entry Criteria

- [ ] PR #84 merged or branch available for testing
- [ ] `post-code-test.sh` executable on target runner environment
- [ ] shellcheck passes on all modified shell scripts

#### II.5 - Risks

- [ ] **Timeline**
  - Risk: None identified; tests are already written and passing.
  - Mitigation: N/A
  - Status: Low

- [ ] **Coverage**
  - Risk: Workflow file detection uses path-based pattern matching only; content-based attacks via non-workflow files are not covered.
  - Mitigation: Document as known limitation; content-based scanning is a separate security concern.
  - Status: Accepted

- [ ] **Environment**
  - Risk: Full post-code pipeline testing requires GitHub Actions runner context (tokens, repo state).
  - Mitigation: Unit tests extract and test logic functions in isolation; CI validates full pipeline.
  - Status: Mitigated

- [ ] **Untestable**
  - Risk: Token permission enforcement is server-side (GitHub); script-level block is the testable layer.
  - Mitigation: Test the script-level gate exhaustively; document that GitHub enforcement is the primary layer.
  - Status: Accepted

- [ ] **Resources**
  - Risk: None; no additional infrastructure or personnel required.
  - Mitigation: N/A
  - Status: Low

- [ ] **Dependencies**
  - Risk: Scaffold embedding must correctly propagate updated post-code.sh to installed repos.
  - Mitigation: Existing scaffold unit tests validate that embedded file changes propagate correctly.
  - Status: Mitigated

- [ ] **Sandbox Token Privilege**
  - Risk: The sandbox `GH_TOKEN` currently grants write permissions (contents:write, issues:write, pull_requests:write). Until a separate read-only sandbox token is minted, the agent operates with more privileges than necessary inside the sandbox.
  - Mitigation: The post-code workflow file block provides defense-in-depth. A follow-up change to mint a read-only sandbox token is tracked as a TODO in the issue.
  - Status: Accepted

- [ ] **Other**
  - Risk: Future changes to post-code.sh section ordering could cause test drift if functions are refactored.
  - Mitigation: Tests use extracted function helpers (not line-number references); resilient to refactoring.
  - Status: Low

---

### Section III - Requirements-to-Tests Mapping

#### III.1 - Requirements Mapping

- **GH-84** | **Workflow file changes are blocked in agent commits (defense-in-depth)**
  - Verify workflow file in `.github/workflows/` is blocked | Functional | P0
  - Verify workflow file among other changed files is blocked | Functional | P0
  - Verify deeply nested workflow file is blocked | Functional | P1
  - Verify `.github/` files outside `workflows/` are not blocked | Functional | P0
  - Verify normal source files are not blocked | Functional | P0
  - Verify empty changed files input passes | Functional | P1
  - Verify blocked output includes file path and blocking reason | Functional | P1
  - Verify error output does not allow GitHub Actions command injection | Functional | P1

- **GH-84** | **Token documentation accurately reflects permissions**
  - Verify code-agent.env comments describe the coder role's actual write permissions | Functional | P1
  - Verify GH_TOKEN comment notes that coder role omits `workflows:write` | Functional | P1

- **GH-84** | **Modified shell scripts pass static analysis**
  - Verify shellcheck produces no new warnings on post-code.sh | Functional | P1

- **GH-84** | **Existing post-code pipeline regression suite passes**
  - Verify all 52 existing post-code tests pass without modification | Regression | P1
  - Verify title rewriting, PR body assembly, no-op detection, push retry, stale branch cleanup, artifact stripping, signed-off-by detection, and error reporting behaviors are preserved | Regression | P1

---

### Section IV - Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
