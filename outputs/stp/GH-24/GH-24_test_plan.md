# Fullsend Test Plan

## **[fix(forge): retry 5xx server errors at the HTTP client level] - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-24](https://github.com/guyoron1/fullsend/issues/24)
- **Feature Tracking:** [GH-24](https://github.com/guyoron1/fullsend/issues/24)
- **Epic Tracking:** [fullsend-ai/fullsend#2342](https://github.com/fullsend-ai/fullsend/pull/2342)
- **QE Owner:** Unassigned
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard STP format. "Verify" = confirm expected behavior; "Validate" = confirm correctness against specification.

### Feature Overview

This change moves 5xx (500-504) server error retry handling from the higher-level `retryOnTransient` wrapper down into the `isRetryable` check within `do()`, the foundational HTTP method used by all GitHub API calls. The former `retryOnTransient` is renamed to `retryOnRepoRace` and narrowed to only handle 404 (async repo init) and 409 (branch ref conflicts). This ensures all GitHub API calls automatically retry on transient server errors, fixing a production 502 Bad Gateway failure where `GetPullRequestHeadSHA` had no retry coverage.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - Upstream PR fullsend-ai/fullsend#2342 provides clear summary of the architectural change: moving 5xx retry from wrapper to `do()` level.
  - Production failure (502 Bad Gateway on `GetPullRequestHeadSHA`) documented as root cause motivation.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - User value: all GitHub API calls gain automatic 5xx retry without developers needing to wrap each call site individually.
  - Customer use case: eliminates intermittent 502/503/504 failures in production workflows.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - All retry behaviors are testable via `httptest.NewServer` mocks returning specific status codes.
  - Clear pass/fail criteria: specific call counts, retry/no-retry decisions, error message content.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - PR provides explicit test plan with 4 test categories covering new retry paths, updated existing tests, and full test suite pass.

- [ ] **Confirmed coverage for NFRs.**
  - Retry backoff timing (exponential: 1s, 2s, 4s) preserves existing non-functional behavior.
  - Max 3 retry attempts in `do()` bounds worst-case latency to ~7s per call.

#### I.2 — Known Limitations

- The `do()` retry loop uses a fixed `maxRetries = 3` constant; this is not configurable at runtime.
- Retry applies uniformly to all 5xx codes (500-504); there is no distinction between idempotent and non-idempotent operations at the `do()` level.
- The comment in `retryOnRepoRace` (line 578) still references "500/502/503/504" in a comment block, though `isTransientStatus` no longer matches those codes. This is a documentation inconsistency in the source but does not affect behavior.

#### I.3 — Technology and Design Review

- [ ] **Developer handoff completed; design and architecture reviewed.**
  - PR authored by @ralphbean, reviewed upstream. Two-file change (`github.go` + `github_test.go`) with clear separation of concerns.

- [ ] **Technology challenges and risks identified.**
  - No new dependencies introduced. Change uses existing `net/http` status code constants.

- [ ] **Test environment needs identified (special hardware, configs, access).**
  - All tests use `httptest.NewServer` — no external dependencies or special environment needed.

- [ ] **API extensions or contract changes reviewed.**
  - No public API changes. `LiveClient` method signatures are unchanged. Only internal retry behavior is modified.

- [ ] **Topology or deployment changes reviewed.**
  - No topology changes. Change is purely in the HTTP client layer.

---

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the retry behavior changes in the `internal/forge/github` package. The scope includes: (1) the addition of 5xx retry in `isRetryable()` consumed by `do()`, (2) the rename and narrowing of `retryOnTransient` to `retryOnRepoRace`, (3) the removal of 5xx codes from `isTransientStatus`, and (4) the elimination of double-retry across the two layers.

**Testing Goals**

- **P0:** Verify `isRetryable()` correctly identifies 5xx (500-504) as retryable and `do()` retries them with proper backoff and exhaustion behavior.
- **P0:** Verify existing rate-limit retry (429, 403 secondary) is not regressed.
- **P1:** Verify `isTransientStatus()` no longer matches 5xx codes and `retryOnRepoRace` only retries 404/409.
- **P1:** Verify file operations (CreateOrUpdateFile, DeleteFile) do not double-retry 5xx at both `do()` and `retryOnRepoRace` layers.
- **P1:** Verify non-retryable status codes (400, 401, 422) pass through without retry.

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Network-level failures (TCP timeouts, DNS errors)** — Handled by Go `net/http` transport layer, not by application-level retry logic.
- [ ] **GitHub API contract validation** — We test retry decisions, not whether GitHub returns correct response payloads.
- [ ] **TLS/authentication failures** — Infrastructure-level concern outside the retry scope.
- [ ] **Performance benchmarking of retry delays** — Backoff timing is deterministic and does not require load testing.

#### II.2 — Test Strategy

**Functional**

- [x] **Functional Testing** — Applicable
  - Verify each status code path in `isRetryable()` and `isTransientStatus()` produces correct retry/no-retry decisions.
- [x] **Automation Testing** — Applicable
  - All scenarios are automated Go unit tests using `httptest.NewServer` and `testify/assert`.
- [x] **Regression Testing** — Applicable
  - Existing 5xx retry tests updated to reflect new call patterns (retry at `do()` level, not wrapper level). Rate-limit tests unchanged.

**Non-Functional**

- [ ] **Performance Testing** — Not Applicable
  - Retry backoff is deterministic; no performance-sensitive changes.
- [ ] **Scale Testing** — Not Applicable
  - Single-request retry logic; no concurrency or scale dimensions.
- [ ] **Security Testing** — Not Applicable
  - No authentication, authorization, or data handling changes.
- [ ] **Usability Testing** — Not Applicable
  - No user-facing interface changes.
- [ ] **Monitoring** — Not Applicable
  - No observability changes; error messages updated but not metrics.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Not Applicable
  - Internal refactor; no external API contract changes.
- [ ] **Upgrade Testing** — Not Applicable
  - No version migration or state persistence changes.
- [ ] **Dependencies** — Not Applicable
  - No new dependencies introduced.
- [ ] **Cross Integrations** — Not Applicable
  - Change is internal to the GitHub client; no cross-component integration needed.

**Infrastructure**

- [ ] **Cloud Testing** — Not Applicable
  - All tests run with mocked HTTP servers; no cloud resources needed.

#### II.3 — Test Environment

- **Cluster Topology:** Not required — unit tests only
- **Platform Version:** Go 1.26.0 (per go.mod)
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** N/A
- **Network:** Loopback only (httptest.NewServer)
- **Operators:** N/A
- **Platform:** Linux (CI)
- **Special Configs:** None

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go test toolchain (`go test`, `httptest`, `testify`).

#### II.4 — Entry Criteria

- [ ] PR branch compiles without errors (`go build ./internal/forge/...`)
- [ ] Existing test suite passes before applying changes (`go test ./internal/forge/github/...`)
- [ ] Code review completed on upstream PR fullsend-ai/fullsend#2342

#### II.5 — Risks

- [ ] **Timeline**
  - Risk: None identified — all tests are unit-level and fast to execute.
  - Mitigation: N/A
  - Status: Low

- [ ] **Coverage**
  - Risk: `do()` serves 20+ call sites via helper methods (get/post/put/patch/delete_); a subtle regression could affect any GitHub API operation.
  - Mitigation: Test each 5xx code individually; verify call counts to detect double-retry.
  - Status: Mitigated

- [ ] **Environment**
  - Risk: None — tests use mocked HTTP servers.
  - Mitigation: N/A
  - Status: Low

- [ ] **Untestable**
  - Risk: Real GitHub 5xx errors are non-deterministic and cannot be reproduced on demand.
  - Mitigation: Mock-based testing validates the decision logic; production monitoring covers real-world behavior.
  - Status: Accepted

- [ ] **Resources**
  - Risk: None — no special resources required.
  - Mitigation: N/A
  - Status: Low

- [ ] **Dependencies**
  - Risk: None — no new external dependencies.
  - Mitigation: N/A
  - Status: Low

- [ ] **Other**
  - Risk: Comment in `retryOnRepoRace` (line 578) still references 5xx codes despite `isTransientStatus` no longer matching them. Could confuse future contributors.
  - Mitigation: Flag for documentation cleanup in code review.
  - Status: Open

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **Requirement ID:** GH-24
- **Requirement Summary:** `do()` retries on 5xx server errors (500-504) automatically for all GitHub API calls
- **Test Scenarios:**
  - Verify `isRetryable` returns true for 500, 502, 503, 504
  - Verify `do()` retries a 502 and succeeds on next attempt
  - Verify `do()` exhausts retries on persistent 5xx and returns "retryable error after 3 attempts"
  - Verify non-5xx server errors (e.g., 505, 511) are not retried
- **Tier:** Unit Tests
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** Rate limit retry (429, 403 secondary) is unaffected by the 5xx refactor
- **Test Scenarios:**
  - Verify 429 Too Many Requests triggers retry in `isRetryable`
  - Verify 403 with "secondary rate limit" body triggers retry
  - Verify 403 without rate limit indicators is not retried
- **Tier:** Unit Tests
- **Priority:** P0

- **Requirement ID:**
- **Requirement Summary:** `retryOnRepoRace` only retries 404 and 409 (5xx removed from `isTransientStatus`)
- **Test Scenarios:**
  - Verify `isTransientStatus` returns true for 404 and 409
  - Verify `isTransientStatus` returns false for 500, 502, 503, 504
  - Verify `retryOnRepoRace` does not retry 5xx errors
- **Tier:** Unit Tests
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** No double-retry of 5xx across `do()` and `retryOnRepoRace` layers
- **Test Scenarios:**
  - Verify CreateOrUpdateFile with 504 on PUT results in exactly 3 HTTP calls (GET, PUT fail, PUT retry succeed) — not 4
  - Verify CreateOrUpdateFile with all 5xx codes follows single-layer retry pattern
  - Verify persistent 5xx exhausts `do()` retries without `retryOnRepoRace` re-attempting
- **Tier:** Unit Tests
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Retry exhaustion produces descriptive error message
- **Test Scenarios:**
  - Verify error contains "retryable error after 3 attempts"
  - Verify error contains HTTP method and path
  - Verify error includes Retry-After header value when present
- **Tier:** Unit Tests
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** Non-retryable HTTP errors pass through immediately without retry
- **Test Scenarios:**
  - Verify 400 Bad Request returns immediately
  - Verify 401 Unauthorized returns immediately
  - Verify 422 Unprocessable Entity returns immediately
  - Verify response body is preserved for non-retryable responses
- **Tier:** Unit Tests
- **Priority:** P1

- **Requirement ID:**
- **Requirement Summary:** File operations (CreateOrUpdateFile, DeleteFile) handle 5xx at `do()` level without re-running preceding GET
- **Test Scenarios:**
  - Verify CreateOrUpdateFile retries PUT at `do()` level without re-executing the GET for file SHA
  - Verify DeleteFile retries at `do()` level
  - Verify CreateOrUpdateFileOnBranch follows same single-layer retry pattern
- **Tier:** Unit Tests
- **Priority:** P1

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | | |
| Dev Lead | | |
| PM | | |
