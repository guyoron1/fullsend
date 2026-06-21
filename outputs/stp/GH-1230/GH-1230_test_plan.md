# Test Plan

## **[GH-1230] Run OutputPipeline on Post-Review Before Posting to Forge - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-1230](https://github.com/fullsend-ai/fullsend/issues/1230)
- **Feature Tracking:** [PR #2444](https://github.com/fullsend-ai/fullsend/pull/2444)
- **Epic Tracking:** Security — Output Sanitization
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QE test plan conventions apply. Priority levels: P0 (must-have), P1 (important), P2 (nice-to-have).

### Feature Overview

This security fix ensures review agent output is sanitized for leaked secrets and obfuscated tokens before being posted to PR comments via the forge API. It applies the existing output sanitization pipeline to the `post-review` CLI command, covering the review body and all finding fields (description and remediation). This closes a gap where the `post-review` code path was the only output channel not already protected by sanitization.

---

### Section I — Motivation & Requirements Review

#### I.1 — Requirement & User Story Review Checklist

- [x] **Reviewed the relevant requirements.**
  - GH-1230 describes a security gap: review agent output was posted to the forge API without secret redaction, risking credential leaks in public PR comments.
  - The fix introduces `sanitizeReviewResult()` which applies the existing `security.OutputPipeline()` to all user-visible text fields before posting.

- [x] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - As a repository owner, I need review agent output to be sanitized so that leaked secrets in agent-generated text are never posted to public PR comments.
  - The user value is preventing accidental credential exposure in automated review comments.

- [x] **Confirmed requirements are **testable and unambiguous**.**
  - Requirements are testable: inject known secret patterns into ReviewResult fields and verify they are redacted after sanitization.
  - The boundary is clear: sanitization occurs between `parseReviewResult` and the forge API calls.

- [x] **Ensured acceptance criteria are **defined clearly**.**
  - AC1: GitHub PATs and API keys in review body are redacted before posting.
  - AC2: Secrets in finding description and remediation fields are redacted.
  - AC3: Zero-width Unicode obfuscation of tokens is detected and redacted.
  - AC4: Clean content without secrets passes through unchanged.

- [x] **Confirmed coverage for NFRs.**
  - Performance: Sanitization adds negligible latency (regex-based string scanning on small text).
  - Security: This IS the security NFR — ensuring no secrets leak through review output.

#### I.2 — Known Limitations

- The `OutputPipeline` relies on pattern-based detection (regex). Novel secret formats not covered by `SecretRedactor` patterns may not be caught.
- Unicode normalization covers known zero-width and bidirectional override characters but may not catch all future obfuscation techniques.
- The sanitization runs in-process on the CLI side; if the forge API is called directly (bypassing the CLI), sanitization is not applied.

#### I.3 — Technology and Design Review

- [x] **Developer handoff completed: architecture and design reviewed.**
  - The implementation follows the established `OutputPipeline` pattern already used in `run.go` and `scan.go`. The `sanitizeReviewResult` function is a pure function operating on `ReviewResult` structs.

- [x] **Technology challenges and mitigations identified.**
  - No new technology challenges. Reuses existing `security.OutputPipeline()` infrastructure (`UnicodeNormalizer` + `SecretRedactor`).

- [x] **Test environment needs identified.**
  - No special environment needed. All tests use `forge.FakeClient` and in-memory structs.

- [x] **API extensions or changes reviewed.**
  - No API changes. The `ReviewResult` struct is unchanged. Sanitization is an internal processing step before existing forge API calls.

- [x] **Topology and deployment considerations reviewed.**
  - N/A — this is a CLI-side processing change with no deployment topology impact.

### Section II — Test Planning

#### II.1 — Scope of Testing

This test plan covers the sanitization of review output in the `post-review` CLI command. The scope includes verifying that the `sanitizeReviewResult` function correctly redacts secrets from review body, finding descriptions, and finding remediations before content reaches the forge API. It also covers verifying that the Unicode normalization step prevents obfuscation-based bypass of secret detection.

**Testing Goals:**

- **P0:** Verify secrets (GitHub PATs, API keys) are redacted from review body and finding fields before forge API calls.
- **P0:** Verify zero-width Unicode obfuscation does not bypass secret redaction.
- **P1:** Verify clean content without secrets passes through unchanged.
- **P1:** Verify sanitization does not break existing post-review flows (approve, request-changes, comment, failure, stale-head).

**Out of Scope (Testing Scope Exclusions):**

- [ ] **SecretRedactor pattern coverage** — The completeness of secret detection patterns is owned by the `security` package and tested separately in `scanner_test.go`.
- [ ] **UnicodeNormalizer correctness** — Unicode normalization logic is owned by the `security` package and tested separately in `unicode_test.go`.
- [ ] **Forge API behavior** — Actual GitHub API responses and error handling are tested in `forge/github/github_test.go`.
- [ ] **Sticky comment posting mechanics** — The `sticky.Post` function is tested separately in the `sticky` package.

#### II.2 — Test Strategy

**Functional:**

- [x] **Functional Testing**
  - Verify `sanitizeReviewResult` correctly processes all ReviewResult fields through the OutputPipeline.
- [x] **Automation Testing**
  - All tests are automated Go unit tests using `testing` + `testify`.
- [x] **Regression Testing**
  - Verify existing post-review flows (approve, request-changes, comment, failure, stale-head) are not broken by the addition of sanitization.
- [ ] **Upgrade Testing**
  - N/A — No upgrade path changes for this security fix.

**Non-Functional:**

- [ ] **Performance Testing**
  - N/A — Regex-based string scanning on small text bodies; no performance concern. For typical review sizes (<10KB), sanitization adds negligible latency. If extremely large reviews (>100KB) become common, performance impact should be revisited.
- [ ] **Scale Testing**
  - N/A — Single review at a time, no scale dimension.
- [x] **Security Testing**
  - Core focus of this change. Verify secret redaction and Unicode obfuscation bypass prevention.
- [ ] **Usability Testing**
  - N/A — No user interface changes.
- [ ] **Monitoring**
  - N/A — No new monitoring or observability changes.

**Integration & Compatibility:**

- [ ] **Compatibility Testing**
  - N/A — No version compatibility concerns.
- [x] **Dependencies**
  - Depends on `security.OutputPipeline()` — `UnicodeNormalizer` and `SecretRedactor`.
- [ ] **Cross Integrations**
  - N/A — Self-contained within the `cli` package.

**Infrastructure:**

- [ ] **Cloud Testing**
  - N/A — No cloud-specific testing needed.

#### II.3 — Test Environment

No special environment needed. All tests are in-process Go unit tests that run on any standard CI runner (Linux) or developer machine (macOS). Requires Go 1.22+ (per go.mod). No cluster, special hardware, network, or storage requirements.

#### II.3.1 — Testing Tools & Frameworks

No new or special tools required. Standard Go testing with testify assertions.

#### II.4 — Entry Criteria

- [ ] `security.OutputPipeline()` is functional and tested (existing `scanner_test.go` passes).
- [ ] `forge.FakeClient` supports all required interface methods for test mocking.
- [ ] `sanitizeReviewResult` function is implemented and compiles.

#### II.5 — Risks

- [ ] **Timeline**
  - Specific Risk: None — tests are straightforward unit tests.
  - Mitigation: N/A
  - Status: [x] Low risk

- [ ] **Coverage**
  - Specific Risk: Novel secret patterns not covered by existing `SecretRedactor` regex may pass through.
  - Mitigation: The `SecretRedactor` pattern library is maintained separately and expanded over time.
  - Status: [x] Accepted — pattern coverage is out of scope for this STP.

- [ ] **Environment**
  - Specific Risk: None — no special environment needed.
  - Mitigation: N/A
  - Status: [x] Low risk

- [ ] **Untestable**
  - Specific Risk: Actual GitHub API posting behavior cannot be tested without integration tests.
  - Mitigation: The `forge.FakeClient` mock verifies the sanitized content reaches the correct API call points.
  - Status: [x] Mitigated

- [ ] **Resources**
  - Specific Risk: None.
  - Mitigation: N/A
  - Status: [x] Low risk

- [ ] **Dependencies**
  - Specific Risk: Changes to `security.OutputPipeline()` behavior could affect sanitization outcomes.
  - Mitigation: `security` package has its own test suite; any behavioral changes would be caught there.
  - Status: [x] Mitigated

- [ ] **Other**
  - Specific Risk: None identified.
  - Mitigation: N/A
  - Status: [x] Low risk

---

### Section III — Requirements-to-Tests Mapping

#### III.1 — Requirements Mapping

- **Requirement ID:** GH-1230
- **Requirement Summary:** Review body content is sanitized for leaked secrets before posting to forge
- **Test Scenarios:**
  - Verify GitHub PAT in review body is redacted (positive)
  - Verify multiple secret types redacted from body (positive)
  - Verify clean body passes through unchanged (positive)
  - Verify body with partial token pattern not over-redacted (negative)
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Edge cases in review body sanitization
- **Test Scenarios:**
  - Verify non-ASCII but non-obfuscation Unicode characters in body pass through unchanged (negative)
- **Tier:** Functional
- **Priority:** P2

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Review finding descriptions and remediations are sanitized for leaked secrets
- **Test Scenarios:**
  - Verify secret redacted from finding description (positive)
  - Verify secret redacted from finding remediation (positive)
  - Verify findings without secrets unchanged (positive)
- **Tier:** Functional
- **Priority:** P0

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Zero-width Unicode obfuscation does not bypass secret redaction
- **Test Scenarios:**
  - Verify zero-width char obfuscated token detected (positive)
  - Verify bidirectional override obfuscation caught (positive)
  - Verify mixed invisible char injection blocked (negative)
- **Tier:** Functional
- **Priority:** P2

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Clean review content passes through sanitization unchanged
- **Test Scenarios:**
  - Verify clean body not modified by sanitization (positive)
  - Verify clean findings not modified by sanitization (positive)
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Mixed empty/non-empty finding fields are sanitized independently
- **Test Scenarios:**
  - Verify finding with empty description but non-empty remediation containing a secret is correctly sanitized (positive)
  - Verify finding with non-empty description containing a secret but empty remediation is correctly sanitized (positive)
  - Verify finding field is preserved when scanner returns empty sanitized result (edge case)
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Empty review body is handled correctly by sanitization
- **Test Scenarios:**
  - Verify empty body skips sanitization scan (positive)
  - Verify failure action with empty body succeeds (positive)
- **Tier:** Functional
- **Priority:** P2

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Posted review content does not contain secrets regardless of input
- **Test Scenarios:**
  - Verify posted PR comment does not contain secrets when review body had secrets (positive)
  - Verify formal review findings posted to PR do not contain secrets (positive)
  - Verify review posted via sticky comment has secrets redacted from body (positive)
- **Tier:** Functional
- **Priority:** P1

---

- **Requirement ID:** GH-1230
- **Requirement Summary:** Existing post-review functionality is not regressed by sanitization
- **Test Scenarios:**
  - Verify approve flow works with sanitization (positive)
  - Verify request-changes flow works with sanitization (positive)
  - Verify comment flow works with sanitization (positive)
  - Verify failure flow works with sanitization (positive)
  - Verify stale-head detection unaffected (positive)
- **Tier:** Functional
- **Priority:** P1

---

### Section IV — Sign-off

| Role | Name | Date |
|:-----|:-----|:-----|
| QE Lead | TBD | |
| Dev Lead | TBD | |
| PM | TBD | |
