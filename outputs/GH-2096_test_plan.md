# Test Plan

## **Two-Pass Review Strategy for Large PRs — Triage Security-Critical Files, Then Deep-Review - Quality Engineering Plan**

### Metadata & Tracking

- **Enhancement:** [GH-2096](https://github.com/fullsend-ai/fullsend/issues/2096)
- **Feature Tracking:** [GH-2096](https://github.com/fullsend-ai/fullsend/issues/2096)
- **Epic Tracking:** [GH-2096](https://github.com/fullsend-ai/fullsend/issues/2096)
- **QE Owner:** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** N/A

**Document Conventions:** Standard QE terminology applies. "Security-critical" refers to files classified by the triage sub-agent based on path patterns and content heuristics. "Uniform attention" means the pre-existing behavior where all files receive equal review context.

### Feature Overview

For PRs exceeding 50 changed files, the review agent now runs a two-pass strategy. A lightweight haiku-model security-triage sub-agent first classifies changed files as security-critical or standard based on path patterns (e.g., `**/mint/**`, `**/auth/**`, `**/oidc/**`) and content heuristics (auth logic, token handling, permission changes). Security-critical files then receive prioritized context in the security and correctness sub-agent context packages, ensuring dedicated reasoning budget rather than competing with boilerplate. This addresses the incident documented in GH-898 where the review agent missed a fail-open security bug on a 52-file PR despite 9 review rounds.

---

### I. Motivation, Requirements & Design

#### I.1 Requirement & User Story Review Checklist

- [ ] **Reviewed the relevant requirements.**
  - GH-2096 specifies the two-pass strategy with threshold-based activation, triage classification, and prioritized context assembly.
  - Related issues GH-898 (parent incident), GH-990 (false safety claims), GH-946 (schema cross-checking) reviewed for context.

- [ ] **Confirmed clear user stories and understood. Understand the value and customer use cases.**
  - Primary use case: large PRs (30+ files, threshold set at 50) where security-critical files are diluted across boilerplate changes.
  - Value: security-critical files get dedicated review context, reducing risk of missed findings like the fail-open bug in GH-898.

- [ ] **Confirmed requirements are **testable and unambiguous**.**
  - Threshold (50 files) is a concrete, testable boundary condition.
  - Classification criteria (path patterns, content heuristics) are enumerated in the sub-agent definition.
  - Fallback behavior (all files treated as security-critical on failure) is explicitly specified.

- [ ] **Ensured acceptance criteria are **defined clearly**.**
  - Triage pass activates at 50+ files; skipped below threshold.
  - Security/correctness sub-agents receive prioritized context; other sub-agents unaffected.
  - Triage failures fall back to uniform attention.
  - New sub-agent excluded from parallel dispatch.

- [ ] **Confirmed coverage for NFRs.**
  - Performance: triage uses haiku model for speed (lightweight classification, not deep reasoning).
  - Reliability: fallback to uniform attention on triage failure ensures no degradation.
  - Maintainability: threshold value is configurable starting point.

#### I.2 Known Limitations

- The 50-file threshold is a starting point and may need tuning based on real-world usage patterns.
- The triage pass uses diff summaries (first ~20 lines per file), not full file content — classification accuracy depends on security signals appearing early in the diff.
- Content heuristics are keyword-based and may produce false positives for files that mention security concepts without implementing them.
- The feature is markdown-only (SKILL.md + sub-agent definition) — no Go code changes, so runtime behavior depends on the agent orchestrator interpreting these specifications correctly.

#### I.3 Technology and Design Review

- [ ] **Developer handoff completed, design and tech overview understood.**
  - PR #2303 reviewed. Changes are confined to two markdown files in the scaffold: SKILL.md orchestrator updates and a new security-triage.md sub-agent definition.
  - Architecture: triage runs synchronously before context assembly (step 3c-1), output feeds step 3d.

- [ ] **Technology challenges identified and understood.**
  - Triage sub-agent output is JSON — parse failures must be handled gracefully.
  - Haiku model classification accuracy for security relevance is unproven at scale.

- [ ] **Test environment needs identified.**
  - No special infrastructure required. Tests operate on scaffold content and orchestrator logic.
  - Triage sub-agent behavior can be tested with mocked PR metadata and file lists.

- [ ] **API extensions and changes reviewed.**
  - No API changes. The sub-agent roster table adds a new entry (security-triage, haiku, pre-pass).
  - Triage output schema: `{ security_critical_files: [{file, reason}], standard_files: [path], summary: string }`.

- [ ] **Topology and special environment requirements reviewed.**
  - No topology requirements. Feature applies to the review agent orchestrator, not cluster infrastructure.

---

### II. Test Planning

#### II.1 Scope of Testing

This test plan covers the two-pass review strategy for large PRs, including: threshold-based activation of the security-triage pre-pass, file classification by path patterns and content heuristics, security-prioritized context package assembly for security and correctness sub-agents, triage failure fallback to uniform attention, sub-agent dispatch exclusion for non-dimension agents, scaffold embedding of the new sub-agent file, and triage output JSON schema validation.

**Testing Goals:**

- **P0:** Verify threshold activation logic correctly gates triage pre-pass at 50-file boundary.
- **P0:** Verify file classification produces correct security-critical vs. standard categorization for known path patterns and content heuristics.
- **P0:** Verify triage failure fallback preserves existing uniform-attention behavior.
- **P1:** Verify security-prioritized context packages are assembled correctly for security and correctness sub-agents.
- **P1:** Verify non-dimension sub-agents are excluded from parallel dispatch loop.
- **P1:** Verify scaffold embedding includes the new security-triage.md file.
- **P1:** Verify triage output JSON schema is correctly parsed and validated.
- **P2:** Verify edge cases (all files critical, no files critical) degrade gracefully.

**Out of Scope (Testing Scope Exclusions):**

- [ ] **Haiku model accuracy benchmarking** -- Classification quality of the haiku model is a model evaluation concern, not a functional test target.
- [ ] **Review quality scoring** -- Measuring whether reviews are objectively "better" with the two-pass strategy is outside functional testing scope.
- [ ] **Performance benchmarking of triage latency** -- Triage speed is expected to be acceptable with haiku; formal latency benchmarks are not in scope.
- [ ] **Downstream repo scaffold installation** -- Testing that `fullsend install` correctly deploys the updated scaffold is covered by existing scaffold installation tests.

#### II.2 Test Strategy

**Functional:**

- [x] **Functional Testing** -- Verify threshold activation, file classification, context assembly, fallback behavior, and dispatch exclusion through unit and functional tests.
- [x] **Automation Testing** -- All tests are automated using Go `testing` + `testify`. No manual test procedures required.
- [x] **Regression Testing** -- Verify existing review behavior is preserved for PRs below the 50-file threshold and when triage fails (fallback path).

**Non-Functional:**

- [ ] **Performance Testing** -- Not applicable. Triage uses haiku model; performance is inherent to model selection.
- [ ] **Scale Testing** -- Not applicable. The feature handles scale by design (triage reduces context for large PRs).
- [x] **Security Testing** -- Verify that security-critical file classification correctly identifies auth, token, permission, and trust boundary files.
- [ ] **Usability Testing** -- Not applicable. Feature is internal to the review agent orchestrator.
- [ ] **Monitoring** -- Not applicable. No new metrics or observability added.

**Integration & Compatibility:**

- [ ] **Compatibility Testing** -- Not applicable. No version-specific behavior.
- [ ] **Upgrade Testing** -- Not applicable. Scaffold files are updated via `fullsend install`.
- [x] **Dependencies** -- Verify the triage sub-agent definition is correctly embedded in the scaffold and accessible via `FullsendRepoFile`.
- [x] **Cross Integrations** -- Verify triage output is correctly consumed by context assembly (step 3d) and does not affect non-security sub-agents.

**Infrastructure:**

- [ ] **Cloud Testing** -- Not applicable. No cloud-specific infrastructure required.

#### II.3 Test Environment

- **Cluster Topology:** N/A — tests run locally, no cluster required
- **Platform Version:** Go 1.26+, fullsend development environment
- **CPU Virtualization:** N/A
- **Compute:** Standard CI runner
- **Special Hardware:** None
- **Storage:** Local filesystem (embedded scaffold content)
- **Network:** N/A — no network-dependent tests
- **Operators:** N/A
- **Platform:** Linux (CI), macOS (local development)
- **Special Configs:** None

#### II.3.1 Testing Tools & Frameworks

No new or special tools required. Standard Go `testing` + `testify/assert` + `testify/require`.

#### II.4 Entry Criteria

- [ ] PR #2303 merged to main branch
- [ ] `go build ./...` succeeds with updated scaffold content
- [ ] Existing scaffold tests (`TestFullsendRepoFilesExist`, `TestCollectInstallFiles_*`) pass
- [ ] `go:embed all:fullsend-repo` correctly includes new `sub-agents/security-triage.md`

#### II.5 Risks

- [ ] **Timeline**
  - Risk: Threshold tuning may require iteration after initial deployment.
  - Mitigation: Threshold is a constant that can be changed in a follow-up PR.
  - Status: [ ] Monitored

- [ ] **Coverage**
  - Risk: Content heuristic false positives may cause unnecessary security-critical classification.
  - Mitigation: False positives are acceptable by design (err on inclusion); false negatives are the real risk.
  - Status: [ ] Accepted

- [ ] **Environment**
  - Risk: None identified. Tests run on standard infrastructure.
  - Mitigation: N/A
  - Status: [x] No risk

- [ ] **Untestable**
  - Risk: Haiku model classification accuracy cannot be deterministically tested — model outputs are non-deterministic.
  - Mitigation: Test the orchestrator's handling of triage output (valid JSON, missing fields, empty response) rather than model accuracy.
  - Status: [ ] Accepted

- [ ] **Resources**
  - Risk: None identified.
  - Mitigation: N/A
  - Status: [x] No risk

- [ ] **Dependencies**
  - Risk: Triage sub-agent depends on Agent tool supporting `model: haiku` and `subagent_type: Explore` parameters.
  - Mitigation: These are existing Agent tool capabilities; no new dependencies introduced.
  - Status: [x] No risk

- [ ] **Other**
  - Risk: Markdown-only changes mean functional behavior depends on agent runtime interpreting SKILL.md correctly.
  - Mitigation: Integration testing of the review agent with a 50+ file PR will validate end-to-end behavior.
  - Status: [ ] Monitored

---

### III. Requirements-to-Tests Mapping

#### III.1 Test Scenarios

- **GH-2096** — Security-triage pre-pass activates for large PRs at file count threshold
  - Verify triage pre-pass runs for PR with >=50 files — Unit Tests — P0
  - Verify triage pre-pass skipped for PR with <50 files — Unit Tests — P0
  - Verify behavior at exact threshold boundary (50 files) — Unit Tests — P0

- **GH-2096** — Security-triage sub-agent classifies files correctly by path patterns and content heuristics
  - Verify mint/auth/oidc paths classified as security-critical — Unit Tests — P0
  - Verify workflow files with permissions blocks classified as security-critical — Unit Tests — P0
  - Verify non-security files classified as standard — Unit Tests — P0
  - Verify ambiguous files default to security-critical — Unit Tests — P0

- **GH-2096** — Security-prioritized context packages assemble correctly
  - Verify security sub-agent receives critical files first — Functional — P1
  - Verify correctness sub-agent receives critical files first — Functional — P1
  - Verify other sub-agents receive standard context — Functional — P1
  - Verify classification headers present in prioritized context — Unit Tests — P1

- **GH-2096** — Triage failure falls back to uniform attention safely
  - Verify fallback on triage sub-agent timeout — Functional — P0
  - Verify fallback on malformed JSON response — Unit Tests — P0
  - Verify fallback on empty triage response — Unit Tests — P0
  - Verify review completes normally after fallback — Functional — P0

- **GH-2096** — Non-dimension sub-agents excluded from parallel dispatch
  - Verify security-triage excluded from step 4 dispatch — Unit Tests — P1
  - Verify challenger excluded from step 4 dispatch — Unit Tests — P1
  - Verify dimension sub-agents dispatched normally — Functional — P1

- **GH-2096** — Scaffold embedding includes new security-triage sub-agent file
  - Verify FullsendRepoFile reads security-triage.md — Unit Tests — P1
  - Verify CollectInstallFiles includes security-triage.md — Unit Tests — P1
  - Verify installed file content matches embedded source — Unit Tests — P1

- **GH-2096** — Triage output JSON schema is valid and consumable
  - Verify valid triage JSON parsed by context assembly — Unit Tests — P1
  - Verify rejection of triage JSON missing required fields — Unit Tests — P1
  - Verify handling of extra unexpected fields in triage JSON — Unit Tests — P1

- **GH-2096** — Edge case: all files security-critical degrades gracefully
  - Verify all-critical classification produces standard-equivalent review — End-to-End — P2
  - Verify no degradation in review quality for all-critical case — End-to-End — P2

- **GH-2096** — Edge case: no files classified as security-critical
  - Verify all files receive standard context when none are critical — Functional — P2
  - Verify triage cost is minimal for zero-critical case — Functional — P2

---

### IV. Sign-off

| Role | Name | Date | Signature |
|:-----|:-----|:-----|:----------|
| QE Lead | TBD | | |
| Dev Lead | TBD | | |
| PM | TBD | | |
