# Test Plan

## GH-79: ADR 0051 — Implement `is_authorized` on All Agent Dispatch Paths

| Field | Value |
|:------|:------|
| **Ticket** | GH-79 |
| **Title** | feat(#1662): ADR 0051 + implement is_authorized on all agent dispatch paths |
| **Product** | fullsend |
| **Author** | QualityFlow |
| **Date** | 2026-06-22 |
| **Status** | Draft |
| **PR** | [#79](https://github.com/guyoron1/fullsend/pull/79) |

---

## I. Introduction

### 1.1 Purpose

This Software Test Plan (STP) defines the test strategy for validating the authorization enforcement changes introduced by ADR 0051. The PR implements `is_authorized` checks on all agent dispatch paths — closing cost-exposure and abuse-surface gaps where previously ungated slash commands (`/fs-triage`, `/fs-code`, `/fs-review`) and PR-triggered auto-review allowed any GitHub user to trigger agent inference runs.

### 1.2 Scope

**In scope:**

- Authorization enforcement on all `/fs-*` slash commands in `reusable-dispatch.yml`
- PR-triggered dispatch (`pull_request_target` opened/synchronize/ready_for_review) author association checks via `is_event_actor_authorized()`
- Preservation of ungated auto-triage on `issues.opened/edited` (ADR 0051 exception)
- Bot user blocking (COMMENT_USER_TYPE != "Bot" short-circuit)
- Label-based bot-to-bot dispatch workflow preservation
- Needs-info re-triage authorization rules (issue author or non-NONE association)
- CLI infrastructure changes (config, forge, harness, binary, dispatch packages)

**Out of scope:**

- Per-user rate limiting for auto-triage (deferred to #1687)
- Visible feedback mechanism for unauthorized users (implementation detail, not tested here)
- GitHub Actions workflow YAML syntax validation (platform-level)
- Go module dependency resolution (build toolchain)

### 1.3 References

| Document | Location |
|:---------|:---------|
| ADR 0051 | `docs/ADRs/0051-require-authorization-on-all-agent-dispatch-paths.md` |
| Dispatch workflow | `.github/workflows/reusable-dispatch.yml` |
| Upstream issue | fullsend-ai/fullsend#1688 |
| Rate limiting followup | fullsend-ai/fullsend#1687 |

---

## II. Test Strategy

### 2.1 Approach

Testing follows a functional verification approach focused on the dispatch routing logic in `reusable-dispatch.yml`. The authorization checks are shell functions (`is_authorized`, `is_event_actor_authorized`) evaluated during the GitHub Actions `route` job. Tests verify correct stage assignment (or non-assignment) based on actor association, user type, and event type.

The CLI and infrastructure changes (100 files, 17909 additions) are covered by existing unit tests in the repository (21 test files modified in this PR). This STP focuses on the authorization behavior that is the core security change.

### 2.2 Test Classification

| Classification | Description | Count |
|:---------------|:------------|:------|
| **Functional** | Authorization logic, dispatch routing, association checks | 34 |
| **E2E** | Agent run pipeline with updated infrastructure | 3 |
| **Total** | | **37** |

### 2.3 Risk Assessment

| Risk | Severity | Mitigation |
|:-----|:---------|:-----------|
| Authorized users blocked from dispatching | High | Test all three valid associations (OWNER, MEMBER, COLLABORATOR) for each command |
| Auto-triage broken for external contributors | High | Explicit test that issues.opened remains ungated |
| Bot-to-bot handoff broken | High | Test label-triggered dispatch (ready-to-code, ready-for-review) still works |
| External users can still trigger agent runs via slash commands | Critical | Negative tests for NONE, CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR associations |
| PR auto-review still fires for external PRs | High | Test is_event_actor_authorized rejects non-member PR authors |

---

## III. Requirements-to-Tests Mapping

### 3.1 Slash Command Authorization (P0)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Slash command authorization enforced on all dispatch paths | Verify unauthorized user cannot trigger /fs-triage | Negative | P0 |
| | | Verify unauthorized user cannot trigger /fs-code | Negative | P0 |
| | | Verify unauthorized user cannot trigger /fs-review | Negative | P0 |
| | | Verify COLLABORATOR can trigger all slash commands | Positive | P0 |
| | | Verify NONE association rejected for all commands | Negative | P0 |
| | | Verify FIRST_TIME_CONTRIBUTOR association rejected | Negative | P0 |

**Evidence:** `reusable-dispatch.yml` — `/fs-triage`, `/fs-code`, `/fs-review` now gated by `is_authorized()` with same pattern as `/fs-fix`, `/fs-retro`, `/fs-prioritize`.

### 3.2 PR-Triggered Dispatch Authorization (P0)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | PR-triggered dispatch checks author_association | Verify MEMBER PR author triggers auto-review | Positive | P0 |
| | | Verify external PR author blocked from auto-review | Negative | P0 |
| | | Verify synchronize event checks PR author association | Positive | P0 |
| | | Verify ready_for_review event checks PR author association | Positive | P0 |

**Evidence:** `reusable-dispatch.yml` — `pull_request_target` opened/synchronize/ready_for_review paths call `is_event_actor_authorized(PR_AUTHOR_ASSOC)`.

### 3.3 Authorized User Dispatch (P0)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Authorized users can dispatch all agent stages | Verify OWNER dispatches all slash commands | Positive | P0 |
| | | Verify MEMBER dispatches all slash commands | Positive | P0 |
| | | Verify COLLABORATOR dispatches all slash commands | Positive | P0 |
| | | Verify /fs-code blocked when PR already exists | Negative | P0 |

**Evidence:** `reusable-dispatch.yml` — OWNER/MEMBER/COLLABORATOR associations pass `is_authorized()` check for all `/fs-*` commands.

### 3.4 Auto-Triage Exception (P1)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Auto-triage on issues.opened/edited remains ungated | Verify any user opening issue triggers triage | Positive | P1 |
| | | Verify issue edit by external user triggers triage | Positive | P1 |
| | | Verify NONE association user triggers auto-triage | Positive | P1 |

**Evidence:** `reusable-dispatch.yml` — issues opened/edited path sets `STAGE=triage` without authorization check (ADR 0051 exception for drive-by bug reporters).

### 3.5 Bot-to-Bot Label Workflows (P1)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Label-based dispatch workflows unaffected | Verify ready-to-code label triggers code dispatch | Positive | P1 |
| | | Verify ready-for-review label triggers review dispatch | Positive | P1 |
| | | Verify label dispatch bypasses is_authorized check | Positive | P1 |

**Evidence:** `reusable-dispatch.yml` — `issues.labeled` path (ready-to-code, ready-for-review) has no `is_authorized` check; label application requires write access (implicit authorization gate).

### 3.6 Bot User Blocking (P1)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Bot users cannot invoke slash commands | Verify Bot user blocked from slash commands | Negative | P1 |
| | | Verify Bot check precedes authorization check | Negative | P1 |
| | | Verify bot-suffix user login handled correctly | Negative | P1 |

**Evidence:** `reusable-dispatch.yml` — `COMMENT_USER_TYPE != "Bot"` check short-circuits before `is_authorized` for all slash command paths.

### 3.7 Authorization Helper Functions (P1)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | is_authorized helper correctly evaluates association | Verify is_authorized accepts OWNER association | Positive | P1 |
| | | Verify is_authorized accepts MEMBER association | Positive | P1 |
| | | Verify is_authorized accepts COLLABORATOR association | Positive | P1 |
| | | Verify is_authorized rejects CONTRIBUTOR association | Negative | P1 |
| | | Verify is_event_actor_authorized with empty association | Negative | P1 |

**Evidence:** `reusable-dispatch.yml` — `is_authorized()` checks `COMMENT_AUTHOR_ASSOC`; `is_event_actor_authorized()` checks passed association parameter. Both use case-statement matching OWNER|MEMBER|COLLABORATOR.

### 3.8 Needs-Info Re-Triage (P2)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | Needs-info re-triage allows authors and non-NONE | Verify issue author re-triggers triage on needs-info | Positive | P2 |
| | | Verify CONTRIBUTOR comment triggers needs-info triage | Positive | P2 |
| | | Verify NONE non-author blocked from needs-info triage | Negative | P2 |
| | | Verify feature-labeled issues skip needs-info triage | Negative | P2 |

**Evidence:** `reusable-dispatch.yml` — default case for `issue_comment` checks `COMMENT_AUTHOR_ASSOC != "NONE"` OR `is_issue_author` for issues with `needs-info` label but not `feature` label.

### 3.9 CLI Infrastructure Compatibility (P1)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | CLI and infrastructure changes preserve agent pipeline | Verify agent run pipeline completes successfully | Positive | P1 |
| | | Verify harness loading with updated config structure | Positive | P1 |
| | | Verify forge.Client interface compatibility | Positive | P1 |

**Evidence:** LSP analysis — `runAgent()` called by `newRunCmd` and 11 test functions; `forge.Client` interface referenced by 36 files across the codebase; `config.ValidRoles()` used in `mint_setup.go` and `config_test.go`.

### 3.10 PR Retro Dispatch (P2)

| Req ID | Requirement | Test Scenario | Type | Priority |
|:-------|:------------|:--------------|:-----|:---------|
| GH-79 | PR retro dispatch on closure ungated | Verify PR closure triggers retro unconditionally | Positive | P2 |
| | | Verify external user PR merge triggers retro | Positive | P2 |

**Evidence:** `reusable-dispatch.yml` — `pull_request_target` closed event sets `STAGE="retro"` unconditionally; merged PR retro is always safe since the merge itself requires write access.

---

## IV. Regression Analysis

### 4.1 LSP Call Graph Analysis

LSP analysis was performed on the Go source code to identify impacted components:

| Symbol | File | References | Impact |
|:-------|:-----|:-----------|:-------|
| `forge.Client` (interface) | `internal/forge/forge.go:166` | 115 references across 36 files | Core abstraction; changes to `forge.Client` interface methods affect all consumers |
| `runAgent` (function) | `internal/cli/run.go:120` | 13 incoming calls (1 production, 12 tests) | Main agent execution path; infrastructure changes here affect all agent runs |
| `config.ValidRoles` (function) | `internal/config/config.go:93` | 5 references across 3 files | Role validation used during mint setup and config validation |
| `bootstrapCommon` (function) | `internal/cli/run.go:995` | 2 references in run.go | Sandbox setup; changes affect all agent sandboxes |

### 4.2 Impacted Components

| Component | Files Changed | Impact Area |
|:----------|:--------------|:------------|
| Dispatch routing | 1 (reusable-dispatch.yml) | Authorization enforcement — **primary change** |
| CLI commands | 10 (admin, mint, run, vendor, etc.) | Command infrastructure — refactoring, new commands |
| Forge interface | 3 (forge.go, fake.go, github.go) | Git forge abstraction — new methods, fake implementation |
| Config | 1 (config.go) | Organization configuration — new fields, validation |
| Harness | 3 (discover_remote.go, harness.go, lint.go) | Agent harness loading — new discovery, linting |
| Binary management | 4 (acquire.go, download.go, vendorroot.go, etc.) | Binary acquisition — download, vendor root |
| GCF provisioner | 3 (fakeclient.go, handler.go.embed, provisioner.go) | Token mint dispatch — handler changes |
| GitHub workflows | 12 files | CI/CD infrastructure — authorization, sandbox images |
| Tests | 21 test files | Test coverage for all above changes |

### 4.3 Dependency Chains

```
reusable-dispatch.yml
  └── is_authorized() ← COMMENT_AUTHOR_ASSOC (issue_comment events)
  └── is_event_actor_authorized() ← PR_AUTHOR_ASSOC (pull_request_target events)
  └── is_issue_author() ← COMMENT_USER_LOGIN == ISSUE_USER_LOGIN
  └── has_label() ← ISSUE_LABELS / PR_LABELS CSV parsing

internal/cli/run.go::runAgent()
  └── harness.LoadWithBase() → harness loading pipeline
  └── bootstrapCommon() → sandbox setup
  └── bootstrapEnv() → environment injection
  └── forge.Client → GitHub API operations (115 refs across 36 files)

internal/config/config.go::ValidRoles()
  └── OrgConfig.Validate() → role validation
  └── mint_setup.go → mint provisioning
```

---

## V. Test Environment

| Component | Specification |
|:----------|:-------------|
| **Platform** | GitHub Actions (ubuntu-latest) |
| **Language** | Go 1.26.0 |
| **Test Framework** | `testing` + `testify` (assert, require) |
| **Dispatch Testing** | Shell script unit tests or workflow simulation |
| **CI Workflow** | `reusable-dispatch.yml` dispatch routing |

---

## VI. Test Summary

| Category | P0 | P1 | P2 | Total |
|:---------|:---|:---|:---|:------|
| Slash command auth | 6 | — | — | 6 |
| PR-triggered auth | 4 | — | — | 4 |
| Authorized user dispatch | 4 | — | — | 4 |
| Auto-triage exception | — | 3 | — | 3 |
| Bot-to-bot labels | — | 3 | — | 3 |
| Bot user blocking | — | 3 | — | 3 |
| Auth helper functions | — | 5 | — | 5 |
| Needs-info re-triage | — | — | 4 | 4 |
| CLI infrastructure | — | 3 | — | 3 |
| PR retro dispatch | — | — | 2 | 2 |
| **Total** | **14** | **17** | **6** | **37** |

---

## VII. Approval

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| Author | QualityFlow | 2026-06-22 | — |
| Reviewer | | | |
| Approver | | | |
