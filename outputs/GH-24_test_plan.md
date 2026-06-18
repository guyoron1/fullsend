# Test Plan

## **Retry 5xx Server Errors at the HTTP Client Level - Quality Engineering Plan**

### **Metadata & Tracking**

- **Enhancement(s):** [GH-24](https://github.com/guyoron1/fullsend/pull/24) — Mirror of [fullsend-ai/fullsend#2342](https://github.com/fullsend-ai/fullsend/pull/2342)
- **Feature Tracking:** [GH-24](https://github.com/guyoron1/fullsend/pull/24)
- **Epic Tracking:** N/A
- **QE Owner(s):** TBD
- **Owning SIG:** N/A
- **Participating SIGs:** None

**Document Conventions (if applicable):** N/A

### **Feature Overview**

This change moves HTTP 5xx (500–504) retry handling from the higher-level `retryOnTransient` wrapper down into the `isRetryable` check within `do()`, so that **all** GitHub API calls automatically retry on transient server errors. The former `retryOnTransient` is renamed to `retryOnRepoRace` and narrowed to only handle 404/409 (repo initialization races and branch ref conflicts). This fix addresses a production 502 Bad Gateway failure and ensures consistent retry behavior across all 19+ callers of `do()`.

---

### **I. Motivation and Requirements Review (QE Review Guidelines)**

This section documents the mandatory QE review process. The goal is to understand the feature's value,
technology, and testability before formal test planning.

#### **1. Requirement & User Story Review Checklist**

- [ ] **Review Requirements**
  - Reviewed the relevant requirements.
  - PR description and upstream PR (fullsend-ai/fullsend#2342) clearly describe the retry boundary change: 5xx errors move from `retryOnTransient` to `do()` via `isRetryable`.
- [ ] **Understand Value and Customer Use Cases**
  - Confirmed clear user stories and understood.
  - Understand the difference between community and product requirements.
  - **What is the value of the feature for customers**.
  - Ensured requirements contain relevant **customer use cases**.
  - Customers experienced 502 Bad Gateway failures in production. This fix ensures all GitHub API operations (repo creation, file management, branch operations, issue management, PR operations) automatically retry on transient server errors without requiring callers to implement their own retry logic.
- [ ] **Testability**
  - Confirmed requirements are **testable and unambiguous**.
  - Retry behavior is fully testable with mock HTTP servers (`httptest.NewServer`). Each status code (500–504) can be injected and retry counts verified. The existing test suite already uses this pattern.
- [ ] **Acceptance Criteria**
  - Ensured acceptance criteria are **defined clearly** (clear user stories; product requirements clearly defined in Jira).
  - AC1: `isRetryable()` returns true for status codes 500, 501, 502, 503, 504. AC2: `do()` retries on 5xx with exponential backoff. AC3: `isTransientStatus()` no longer includes 5xx codes. AC4: `retryOnRepoRace` handles only 404/409. AC5: Error message reads "retryable error" instead of "rate limited".
- [ ] **Non-Functional Requirements (NFRs)**
  - Confirmed coverage for NFRs, including Performance, Security, Usability, Downtime, Connectivity, Monitoring (alerts/metrics), Scalability, Portability (e.g., cloud support), and Docs.
  - Performance: Retry backoff timing must not cause excessive delays (exponential backoff capped at `maxRetries`). Reliability: No double-retry when `retryOnRepoRace` wraps `do()` for the same 5xx error.

#### **2. Known Limitations**

- 5xx retry in `do()` drains the response body (`io.Copy(io.Discard, resp.Body)`) — the original response body content is lost on retry, which is acceptable since 5xx responses rarely contain actionable data.
- The retry applies uniformly to all 5xx codes (500–504). Status codes outside this range (e.g., 505 HTTP Version Not Supported) are not retried, which is intentional.
- `retryOnRepoRace` no longer retries 5xx errors directly; it relies on `do()` for that. If a higher-level operation receives a `do()`-level exhaustion error for 5xx, `retryOnRepoRace` will not add additional retry attempts for that class of failure.

#### **3. Technology and Design Review**

- [ ] **Developer Handoff/QE Kickoff**
  - A meeting where Dev/Arch walked QE through the design, architecture, and implementation details. **Critical for identifying untestable aspects early.**
  - PR author (@ralphbean) provided detailed description of the retry boundary change. The architectural decision to push 5xx retries into `do()` ensures uniform coverage across all 19 callers (identified via LSP `incomingCalls` analysis).
- [ ] **Technology Challenges**
  - Identified potential testing challenges related to the underlying technology.
  - Testing must verify no double-retry: when `retryOnRepoRace` wraps a `do()` call that encounters a 5xx, the retry should happen only at the `do()` level. Mock HTTP servers must carefully count call sequences to distinguish single-level vs. double-level retries.
- [ ] **Test Environment Needs**
  - Determined necessary **test environment setups and tools**.
  - Standard Go test environment with `net/http/httptest` mock servers. No external services or clusters required.
- [ ] **API Extensions**
  - Reviewed new or modified APIs and their impact on testing.
  - No new APIs. The `isRetryable()` function signature is unchanged. `retryOnTransient` is renamed to `retryOnRepoRace` (internal, unexported). Error message format changed from "rate limited" to "retryable error".
- [ ] **Topology Considerations**
  - Evaluated multi-cluster, network topology, and architectural impacts.
  - Not applicable — this is an HTTP client-level change with no infrastructure topology impact.

---

### **II. Software Test Plan (STP)**

This STP serves as the **overall roadmap for testing**, detailing the scope, approach, resources, and schedule.

#### **1. Scope of Testing**

Testing covers the modified retry logic in the GitHub HTTP client layer (`internal/forge/github/github.go`). The scope includes: (1) `isRetryable()` correctly identifying 5xx status codes as retryable, (2) `do()` performing retries with correct backoff on 5xx errors, (3) `retryOnRepoRace()` (formerly `retryOnTransient`) handling only 404/409, (4) `isTransientStatus()` excluding 5xx codes, (5) error message format changes, and (6) no double-retry behavior when both retry layers are involved.

**Testing Goals**

- **P0:** Verify `isRetryable()` correctly identifies all 5xx status codes (500–504) as retryable and returns `(true, nil)`.
- **P0:** Verify `do()` retries on 5xx errors and eventually succeeds or exhausts retries with correct error message.
- **P0:** Verify no double-retry when `retryOnRepoRace` wraps a `do()` call encountering 5xx.
- **P1:** Verify `isTransientStatus()` returns false for 5xx codes (only true for 404/409).
- **P1:** Verify `retryOnRepoRace` still correctly retries on 404 and 409.
- **P1:** Verify error message format is "retryable error after N attempts" (not "rate limited").
- **P2:** Verify non-retryable status codes (e.g., 400, 401, 422) are not retried.

**Out of Scope (Testing Scope Exclusions)**

- [ ] **Rate limit retry logic (429 and 403 with Retry-After)** — Not modified in this PR; existing tests cover this. Agreement: N/A
- [ ] **GitHub API endpoint correctness** — API path construction and response parsing are unchanged. Agreement: N/A
- [ ] **Network-level failures (DNS, TCP, TLS)** — `http.Client.Do()` errors are not retried by `do()` and are unchanged. Agreement: N/A
- [ ] **End-to-end integration with live GitHub API** — Unit tests with mock servers provide sufficient coverage for retry logic. Agreement: N/A

#### **2. Test Strategy**

**Functional**

- [ ] **Functional Testing** — Validates that the feature works according to specified requirements and user stories
  - *Details:* Unit tests with mock HTTP servers verify each 5xx status code triggers retry in `isRetryable()` and `do()`. Tests verify correct call counts, backoff behavior, and error messages.
- [ ] **Automation Testing** — Confirms test automation plan is in place for CI and regression coverage (all tests are expected to be automated)
  - *Details:* All tests are Go unit tests (`go test`) running in CI. No manual testing required.
- [ ] **Regression Testing** — Verifies that new changes do not break existing functionality
  - *Details:* Existing tests for rate limiting (429/403), file operations, and repo race retries must continue passing. The existing `TestCreateOrUpdateFile_RetriesOn504` test is updated to reflect the new retry boundary.

**Non-Functional**

- [ ] **Performance Testing** — Validates feature performance meets requirements (latency, throughput, resource usage)
  - *Details:* Not applicable — retry delays are deterministic (exponential backoff) and already bounded by `maxRetries`.
- [ ] **Scale Testing** — Validates feature behavior under increased load and at production-like scale
  - *Details:* Not applicable — retry logic is per-request, not scale-dependent.
- [ ] **Security Testing** — Verifies security requirements, RBAC, authentication, authorization, and vulnerability scanning
  - *Details:* Not applicable — no authentication or authorization changes.
- [ ] **Usability Testing** — Validates user experience and accessibility requirements
  - *Details:* Not applicable — internal HTTP client, no user-facing interface.
- [ ] **Monitoring** — Does the feature require metrics and/or alerts?
  - *Details:* Not applicable — no new metrics or alerts. Error messages are updated but monitoring is unchanged.

**Integration & Compatibility**

- [ ] **Compatibility Testing** — Ensures feature works across supported platforms, versions, and configurations
  - *Details:* Not applicable — pure Go code with no platform-specific dependencies.
- [ ] **Upgrade Testing** — Validates upgrade paths from previous versions, data migration, and configuration preservation
  - *Details:* Not applicable — no persistent state or configuration changes.
- [ ] **Dependencies** — Blocked by deliverables from other components/products
  - *Details:* No external dependencies. Uses only stdlib `net/http`.
- [ ] **Cross Integrations** — Does the feature affect other features or require testing by other teams?
  - *Details:* All 19 callers of `do()` inherit the new 5xx retry behavior. Callers include: `get`, `post`, `put`, `patch`, `delete_`, `GetRepo`, `CreateOrUpdateFile`, `CreateOrUpdateFileOnBranch`, `DeleteFile`, `GetTokenScopes`, `RepoSecretExists`, `RepoVariableExists`, `GetRepoVariable`, `DispatchWorkflow`, `GetWorkflowRunLogs`, `OrgSecretExists`, `DeleteOrgSecret`, `OrgVariableExists`, `DeleteOrgVariable`. Regression coverage for file operations (which also use `retryOnRepoRace`) is critical.

**Infrastructure**

- [ ] **Cloud Testing** — Does the feature require multi-cloud platform testing?
  - *Details:* Not applicable — HTTP client logic is cloud-agnostic.

#### **3. Test Environment**

- **Cluster Topology:** Not applicable — unit tests only
- **Platform & Product Version(s):** Go 1.26+, as specified in go.mod
- **CPU Virtualization:** Not applicable
- **Compute Resources:** Standard CI runner
- **Special Hardware:** None
- **Storage:** None
- **Network:** `net/http/httptest` mock servers (localhost)
- **Required Operators:** None
- **Platform:** Linux (CI), macOS/Linux (local development)
- **Special Configurations:** None

#### **3.1. Testing Tools & Frameworks**

- **Test Framework:** Go standard `testing` package + `github.com/stretchr/testify` (assert/require)
- **CI/CD:** Standard (no special tools)
- **Other Tools:** `net/http/httptest` for mock HTTP servers

#### **4. Entry Criteria**

The following conditions must be met before testing can begin:

- [ ] Requirements and design documents are **approved and merged**
- [ ] Test environment can be **set up and configured** (see Section II.3 - Test Environment)
- [ ] PR code changes are available on the test branch (`fix/retry-5xx-in-do`)
- [ ] Go module dependencies are resolved (`go mod download`)

#### **5. Risks**

- [ ] **Timeline/Schedule**
  - Risk: Low — all tests are unit tests with fast execution time.
  - Mitigation: Tests can be written and run in a single sprint.
- [ ] **Test Coverage**
  - Risk: Medium — 19 callers of `do()` inherit the new retry behavior; not all callers have dedicated 5xx retry tests.
  - Mitigation: Focus on `isRetryable()` unit tests (covering all 5xx codes) plus integration tests for key callers (`CreateOrUpdateFile`, `putFileWithRetry`). The `do()` retry loop is the single enforcement point, so testing it directly provides coverage for all callers.
- [ ] **Test Environment**
  - Risk: Low — mock servers are self-contained, no external dependencies.
  - Mitigation: N/A
- [ ] **Untestable Aspects**
  - Risk: Low — the only untestable aspect is real GitHub 5xx behavior timing, which varies in production.
  - Mitigation: Mock servers simulate deterministic 5xx responses. Production monitoring validates real-world retry behavior.
- [ ] **Resource Constraints**
  - Risk: Low — no special resources needed.
  - Mitigation: N/A
- [ ] **Dependencies**
  - Risk: Low — no external team dependencies.
  - Mitigation: N/A
- [ ] **Other**
  - Risk: Double-retry interaction between `retryOnRepoRace` and `do()` could mask failures or cause excessive retries if not properly validated.
  - Mitigation: Dedicated double-retry test verifies that `retryOnRepoRace` does not add additional 5xx retries on top of `do()` retries.

---

### **III. Test Scenarios & Traceability**

This section links requirements to test coverage, enabling reviewers to verify all requirements are tested.

#### **1. Requirements-to-Tests Mapping**

- **Requirement ID:** GH-24
  **Requirement Summary:** `isRetryable()` correctly identifies 5xx server errors as retryable
  **Test Scenarios:**
  - Verify `isRetryable` returns true for HTTP 500 (Internal Server Error) — *positive*
  - Verify `isRetryable` returns true for HTTP 502 (Bad Gateway) — *positive*
  - Verify `isRetryable` returns true for HTTP 503 (Service Unavailable) — *positive*
  - Verify `isRetryable` returns true for HTTP 504 (Gateway Timeout) — *positive*
  - Verify `isRetryable` returns true for HTTP 501 (Not Implemented) — *positive*
  - Verify `isRetryable` drains response body on 5xx — *positive*
  - Verify `isRetryable` returns false for non-retryable codes (400, 401, 404, 422) — *negative*
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** `do()` retries HTTP requests on 5xx server errors with exponential backoff
  **Test Scenarios:**
  - Verify `do()` retries and succeeds after transient 502 — *positive*
  - Verify `do()` retries and succeeds after transient 503 — *positive*
  - Verify `do()` exhausts retries and returns error after persistent 500 — *negative*
  - Verify exhaustion error message reads "retryable error after N attempts" — *positive*
  - Verify `do()` respects context cancellation during retry backoff — *negative*
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** No double-retry when `retryOnRepoRace` wraps `do()` for 5xx errors
  **Test Scenarios:**
  - Verify `CreateOrUpdateFile` with 504 retries only at `do()` level (3 total calls: GET, PUT fail, PUT succeed) — *positive*
  - Verify `CreateOrUpdateFile` with all 5xx codes retries only at `do()` level — *positive*
  - Verify `retryOnRepoRace` does not re-invoke operation on `do()`-exhausted 5xx error — *negative*
  **Tier:** Unit Tests
  **Priority:** P0

- **Requirement ID:**
  **Requirement Summary:** `retryOnRepoRace` handles only 404 and 409 repo race conditions
  **Test Scenarios:**
  - Verify `retryOnRepoRace` retries on 404 (async repo init) — *positive*
  - Verify `retryOnRepoRace` retries on 409 (branch ref conflict) — *positive*
  - Verify `retryOnRepoRace` does not retry on 500 — *negative*
  - Verify `retryOnRepoRace` does not retry on 502 — *negative*
  - Verify `retryOnRepoRace` exhausts attempts and returns wrapped error — *negative*
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** `isTransientStatus` returns true only for 404 and 409
  **Test Scenarios:**
  - Verify `isTransientStatus` returns true for 404 — *positive*
  - Verify `isTransientStatus` returns true for 409 — *positive*
  - Verify `isTransientStatus` returns false for 500, 502, 503, 504 — *negative*
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** Error messages accurately reflect retry exhaustion reason
  **Test Scenarios:**
  - Verify error message includes "retryable error" (not "rate limited") — *positive*
  - Verify error message includes method, path, and delay information — *positive*
  - Verify error message includes Retry-After header value when present — *positive*
  **Tier:** Unit Tests
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** File operations correctly use `retryOnRepoRace` with narrowed scope
  **Test Scenarios:**
  - Verify `CreateOrUpdateFile` succeeds on first attempt — *positive*
  - Verify `CreateOrUpdateFileOnBranch` retries on 404 via `retryOnRepoRace` — *positive*
  - Verify `DeleteFile` retries on 409 via `retryOnRepoRace` — *positive*
  - Verify `putFileWithRetry` passes through non-transient errors — *negative*
  **Tier:** Functional
  **Priority:** P1

- **Requirement ID:**
  **Requirement Summary:** Rate limit retry behavior is preserved unchanged
  **Test Scenarios:**
  - Verify `isRetryable` still returns true for 429 (Too Many Requests) — *positive*
  - Verify `isRetryable` still returns true for 403 with Retry-After header — *positive*
  - Verify `isRetryable` still detects secondary rate limit in response body — *positive*
  - Verify rate limit backoff timing is unchanged — *positive*
  **Tier:** Unit Tests
  **Priority:** P1

---

### **IV. Sign-off and Approval**

This Software Test Plan requires approval from the following stakeholders:

* **Reviewers:**
  - [TBD / @reviewer]
* **Approvers:**
  - [TBD / @approver]
