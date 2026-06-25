# Test Plan

## **fix(post-code): block workflow file changes and correct token docs - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-84](https://github.com/guyoron1/fullsend/issues/84)
- **Feature Tracking:** [GH-84](https://github.com/guyoron1/fullsend/issues/84)
- **Epic Tracking:** N/A
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** This STP covers security hardening changes to the post-code pipeline. "Post-code" refers to the shell script that runs on the GitHub Actions runner after the sandbox is destroyed, responsible for pushing agent commits and creating PRs. "Workflow file block" refers to the new defense-in-depth gate that rejects agent commits containing `.github/workflows/` modifications.

### Feature Overview

This change hardens the fullsend code agent's post-code pipeline against prompt injection attacks that could cause the agent to create malicious GitHub Actions workflow files. It adds an explicit script-level gate in `post-code.sh` (section 2c) that blocks any agent commit touching `.github/workflows/`, ensuring protection survives future token role changes or GitHub permission model shifts. Additionally, it corrects misleading comments in `code-agent.env` that incorrectly described the sandbox `GH_TOKEN` as read-only when it actually grants write permissions. Six new unit tests validate the workflow file detection logic.

---

### Section I - Motivation & Requirements Review

#### I.1 - Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-84 describes three changes: workflow file blocking in post-code.sh, token documentation correction in code-agent.env, and 6 unit tests for workflow detection logic.
  - The issue references a team sync discussion on code agent security (prompt injection attack vector).

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - The value proposition is security hardening: preventing a prompt-injected agent from pushing malicious workflow files that execute arbitrary code on the org's runner with access to repo secrets.
  - User story: As a fullsend operator, I need defense-in-depth against workflow file injection so that even if the coder token gains `workflows:write` in the future, the script-level gate prevents exploitation.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - The workflow blocking logic is deterministic: any file matching `.github/workflows/*` in the changed files list triggers a hard block (exit 1).
  - Token documentation changes are informational and verifiable by code review.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Acceptance criteria from the issue: all 58 post-code tests pass (including 6 new), shellcheck clean, security self-review approved, CI passes.

- [ ] **Confirmed coverage for NFRs.**
  - Security: defense-in-depth against prompt injection attack vector.
  - Reliability: existing post-code pipeline behavior preserved (regression).

#### I.2 - Known Limitations

- The workflow file block operates on file paths only (pattern matching `.github/workflows/*`). It does not inspect file contents for malicious payloads in non-workflow files.
- The `code-agent.env` token documentation correction is informational only; it does not change any runtime behavior or token permissions.
- The defense-in-depth gate assumes the coder token's `workflows:write` omission remains the primary enforcement; the script-level block is a secondary layer.
- A TODO remains for minting a separate read-only sandbox token (noted in the issue but not implemented in this PR).

#### I.3 - Technology and Design Review

- [ ] **Developer handoff completed.**
  - PR #84 modifies `internal/scaffold/fullsend-repo/scripts/post-code.sh` (new section 2c), `internal/scaffold/fullsend-repo/scripts/post-code-test.sh` (6 new tests), and `internal/scaffold/fullsend-repo/env/code-agent.env` (comment correction).

- [ ] **Technology challenges identified.**
  - Shell script testing relies on function extraction and isolated execution (no git repo or network access needed for tests).
  - The scaffold embed system (`embed.FS` in `internal/scaffold/scaffold.go`) bundles post-code.sh into the Go binary; changes propagate through `FullsendRepoFile()` -> `WalkFullsendRepo()` -> `CollectInstallFiles()` -> `Install()`.

- [ ] **Test environment needs assessed.**
  - Unit tests for workflow detection run in bash without external dependencies.
  - Integration testing of the full post-code pipeline requires a GitHub Actions runner context (PUSH_TOKEN, REPO_FULL_NAME, etc.).

- [ ] **API extensions reviewed.**
  - No API changes. The workflow block is internal to the post-code script.

- [ ] **Topology and deployment considerations reviewed.**
  - The post-code.sh script runs on the GitHub Actions runner (not in the sandbox). Changes affect all repos using fullsend's scaffold system.

### Section II - Test Planning

#### II.1 - Scope of Testing

This test plan covers the security hardening changes to the post-code pipeline: the new workflow file blocking gate, the corrected token documentation, and regression of existing post-code functionality.

**Testing Goals:**

- **P0:** Verify workflow file detection blocks `.github/workflows/` changes with clear error output.
- **P0:** Verify existing post-code pipeline functionality is preserved (title rewriting, PR body assembly, no-op detection, push retry, stale branch cleanup, artifact stripping, signed-off-by detection).
- **P1:** Verify edge cases in workflow detection (nested paths, non-workflow .github/ files, empty input).
- **P1:** Verify scaffold embedding propagates updated scripts correctly.
- **P2:** Verify error reporting comment includes correct workflow run URL.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **GitHub's server-side `workflows:write` enforcement.** Platform-level permission enforcement is owned by GitHub; the script-level block is a defense-in-depth addition.
- [ ] **Sandbox token minting changes.** The TODO for a separate read-only sandbox token is acknowledged but not implemented in this PR.
- [ ] **Prompt injection detection or prevention in the agent itself.** This PR addresses the post-code gate, not the agent's input handling.
- [ ] **Full end-to-end GitHub Actions workflow execution.** The post-code script is tested via extracted function unit tests; full pipeline testing requires CI infrastructure.

#### II.2 - Test Strategy

**Functional:**

- [x] **Functional Testing** -- Core workflow file detection and blocking logic, post-code pipeline regression tests.
  - All 6 new workflow detection tests plus all 52 existing post-code tests.
- [x] **Automation Testing** -- All tests are automated shell scripts executable via `bash post-code-test.sh`.
  - No manual test steps required.
- [x] **Regression Testing** -- Full regression of existing post-code behaviors: title rewriting, PR body assembly, no-op detection, stale branch cleanup, push retry logic, error comment generation, artifact stripping, signed-off-by detection.
  - All existing test functions preserved and passing.

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable; shell script execution is near-instantaneous.
- [ ] **Scale Testing** -- Not applicable; post-code processes a single PR at a time.
- [x] **Security Testing** -- Primary focus of this change. Validates defense-in-depth against workflow file injection attack vector.
  - Verifies blocking behavior for various `.github/workflows/` path patterns.
- [ ] **Usability Testing** -- Not applicable; no user-facing UI changes.
- [ ] **Monitoring** -- Not applicable; no new observability instrumentation.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable; bash script compatible with existing runner environments.
- [ ] **Upgrade Testing** -- Not applicable; scaffold embedding handles propagation.
- [x] **Dependencies** -- Verified scaffold embedding chain: `FullsendRepoFile()` -> `WalkFullsendRepo()` -> `CollectInstallFiles()` -> `Install()`.
  - Go embed.FS in scaffold.go bundles the updated scripts.
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
  - Risk: Scaffold embedding chain must correctly propagate updated post-code.sh to installed repos.
  - Mitigation: Existing scaffold tests (`TestWalkFullsendRepo`, `TestCollectInstallFiles_*`) validate embedding.
  - Status: Mitigated

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
  - Verify blocked output includes clear error message | Functional | P1

- | **Existing post-code title rewriting is preserved (regression)**
  - Verify conventional commit without scope gets issue reference injected | Functional | P0
  - Verify conventional commit with existing scope is not modified | Functional | P0
  - Verify non-conventional title is not modified | Functional | P1
  - Verify various commit types (fix, feat, chore, docs, refactor, test, ci) are handled | Functional | P1

- | **PR body assembly is correct (regression)**
  - Verify PR body contains exactly one Closes line | Functional | P0
  - Verify PR body does not contain Changed files or Created by sections | Functional | P1
  - Verify empty commit body uses fallback description | Functional | P1

- | **No-op detection works correctly (regression)**
  - Verify no-op on main/master branch | Functional | P0
  - Verify no-op on detached HEAD | Functional | P1
  - Verify no-op on feature branch with no changes | Functional | P0
  - Verify proceed on feature branch with changes | Functional | P0

- | **Stale branch cleanup logic is correct (regression)**
  - Verify skip when no remote branch exists | Functional | P1
  - Verify delete when stale branch has no open PR | Functional | P1
  - Verify keep when branch has open PR | Functional | P1

- | **Push retry logic handles errors correctly (regression)**
  - Verify successful push requires no retry | Functional | P1
  - Verify non-fast-forward triggers force-with-lease retry | Functional | P1
  - Verify unexpected error causes failure | Functional | P1

- | **Error reporting comment is correct (regression)**
  - Verify error comment includes exit code and workflow link | Functional | P1
  - Verify org-mode uses dispatch repo URL | Functional | P1
  - Verify non-org-mode falls back to source repo URL | Functional | P1

- | **Agent artifact stripping works correctly (regression)**
  - Verify `.agentready/` files are stripped | Functional | P1
  - Verify `.fullsend-workspace/` files are stripped | Functional | P1
  - Verify normal files are not stripped | Functional | P0
  - Verify multiple artifacts stripped together | Functional | P1

- | **Signed-off-by trailer detection works correctly (regression)**
  - Verify commit with Signed-off-by trailer is blocked | Functional | P0
  - Verify commit without trailer passes | Functional | P0
  - Verify mid-line mention is not falsely detected | Functional | P1
  - Verify case-sensitive detection (lowercase variant passes) | Functional | P1

- | **Token documentation accurately reflects permissions**
  - Verify code-agent.env comments describe correct token permissions | Functional | P1
  - Verify GH_TOKEN comment mentions coder role omits workflows:write | Functional | P1

- | **Scaffold embedding propagates updated scripts**
  - Verify post-code.sh is accessible via FullsendRepoFile() | Functional | P1
  - Verify WalkFullsendRepo includes updated post-code.sh | Functional | P1
  - Verify CollectInstallFiles bundles post-code.sh for installation | Functional | P1

---

### Section IV - Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | | | |
| Dev Lead | | | |
| PM | | | |
