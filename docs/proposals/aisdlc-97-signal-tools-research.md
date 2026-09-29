# Signal tools: what the documentation says (2026-09-29)

Documentation only, nothing installed or run (evidence level `doc` in AISDLC-99's rubric). Four groups, read in
parallel: interface compatibility, dependency classification and supply chain, history and behavior, multi-repo
orchestration. Reach has its own note: [aisdlc-97-reach-tools-research.md](aisdlc-97-reach-tools-research.md).

## Bottom lines

1. **Interface compatibility: for operators, crdify is the right T3 check.** It catches both of our examples:
   an added optional field passes, and a tightened validation pattern is flagged. go-apidiff misses the second one,
   because a `+kubebuilder` validation marker is a comment and the Go API doesn't change.
   - Every tool is blind to CEL rules (`x-kubernetes-validations`), admission and conversion webhooks, defaulting in
     controller code, and behavior. Cover those with a path rule: touching them escalates.
   - For REST, use oasdiff: ERR maps to T3, WARN to T2 or human review.
   - For gRPC, `buf breaking` covers the wire format but not protovalidate rules.
2. **Dependencies: classify from the bot, verify with our own diff.**
   - Renovate's `updateType` or Dependabot's `fetch-metadata` gives patch, minor, major, digest or pin.
   - Our own `go.mod` diff must agree: only versions on existing `require` lines change, the module path stays the
     same, and there's no new `require` and no `replace` change.
   - A bump the bot can't classify never goes to T0. Dependabot returns an empty type for digest and SHA bumps.
3. **"Clean supply-chain checks" can be made concrete.**
   - dependency-review-action passes with its defaults (fails on any vulnerability of low severity or above in
     runtime dependencies), plus a license allowlist.
   - No *new, reachable* vulnerability on head versus base, from govulncheck or OSV-Scanner's `called: true`.
   - The release is at least 3 days old (Renovate `minimumReleaseAge`, or Dependabot's default cooldown).
   - Maintainers haven't changed (Dependabot `maintainer-changes`).
   - GitHub Actions are pinned by SHA.
   - Image digest bumps are invisible to dependency-review, so they need an OSV-Scanner image scan comparing old and
     new.
   - Scorecard is advisory only; its own README warns the aggregate score says little.
4. **Go 0.x versions:** both bots call 0.2→0.3 a minor bump, so it's already T3. A 0.x *patch* can still break, and
   Renovate's own automerge example excludes 0.x. Proposed default: a 0.x patch is not T0.
5. **History: ADR 0089 is not a deterministic source.**
   - Its git-history tier is run by an LLM sub-agent and isn't reproducible, which clashes with decision 2.
   - Its revert check greps commit messages for file names, which rarely appear there.
   - Deterministic replacements:
     - *revert in 90 days:* parse `This reverts commit <sha>` and take both commits' files;
     - *missing co-change:* code-maat `coupling` over 6–12 months, coupling degree of about 50 or more;
     - *hotspot:* above the repository's own 90th percentile of changes per file.
6. **Sensitivity: ADR 0089's "security patterns" are only directory names:** `mint/ auth/ oidc/ rbac/ permissions/
   secrets/ crypto/ token/ tokens/ trust/ policies/`, matched as path substrings. Nothing detects "a risky new import"
   (our T2 rule) or sensitive content.
7. **Behavior: flag flips are detectable only in declarative flag files.**
   - flagd (compare the effective default, not only `defaultVariant`), the OpenFeature manifest, or the Kubernetes
     feature list (`versioned_feature_list.yaml`).
   - Go `pflag`/`cobra` defaults and environment-variable gates need a repository convention (a flag registry) or a
     small code check. No tool was found.
   - "First caller of new code" needs a call-graph diff. No ready tool was found.
8. **Multi-repo PR sets: declare-and-wait is the answer to question 6.**
   - The only two tools that handle dependencies across GitHub repositories (Mergify and Zuul) both use a
     `Depends-On:` line in the PR body and make B wait until A merges.
   - Enforced order would need merge rights in every repository and a coordinator across repositories. It still
     wouldn't be atomic: Zuul's own docs warn partial merges break gating.
   - Add three rules:
     - reject dependency cycles;
     - re-evaluate B when A merges;
     - B's `go.mod` must point at A's default branch or a tag, not A's PR branch.
   - Optional, advisory: test B against A's unmerged PR, as Zuul does.
9. **Our "alternatives considered" claim is accurate, but needs two corrections.**
   - Mergify and Zuul do order merges across repositories via `Depends-On`.
   - Mergify and GitHub rulesets do use changed paths and scopes, so "a fixed check list" is too strong.
   - Still true: none of them computes reach.

## Per group

### Interface compatibility

| Tool | Detects | Our two examples | Output | Limits | Fit |
|---|---|---|---|---|---|
| crdify (kubernetes-sigs, v0.6.0+) | CRD scope, field and version removal, enum narrowing, default changes, tightened min/max, new required fields, type and pattern changes | both correct | text, JSON, YAML; non-zero exit on Error | no CEL, format or uniqueItems; flags any regex change (even a loosened one) and description edits | best for CRD T3 |
| go-apidiff / x/exp/apidiff | exported Go API breaks | misses validation tightening | text; GitHub Action gives semver type | "no tool can detect behavioral changes" | second T3 signal for importable Go modules |
| gorelease | same, plus `go.mod` checks | same | text | never fails on v0 modules; experimental | poor |
| openshift/crd-schema-checker | CRD rules (for example NoBools, field removal) | unverified | error lines | rule list undocumented; "still evolving" | secondary |
| kube-api-linter | Kubernetes API conventions on current types | not a diff tool | golangci-lint findings | no releases yet | quality signal, not T3 |
| oasdiff | hundreds of OpenAPI checks: ERR / WARN / INFO | a pattern added is ERR; a pattern changed is WARN | many formats; `--fail-on` | declared contract only | good for REST |
| `buf breaking` | FILE ⊃ PACKAGE ⊃ WIRE_JSON ⊃ WIRE | misses protovalidate rules | text, JSON, GitHub annotations | custom options out of scope | good for gRPC wire |
| japicmp | Java binary and source compatibility | n/a | reports | bytecode only | Java repositories only |

### Dependencies and supply chain

| Tool | Role | Key facts | Limits |
|---|---|---|---|
| Renovate | classification | `updateType`: major, minor, patch, pin, pinDigest, digest, lockFileMaintenance, rollback, bump, replacement; expose it via a `{{updateType}}` label | 0.x minor stays "minor"; no labels unless configured; `minimumReleaseAge` can hold Go pseudo-versions forever |
| Dependabot `fetch-metadata` | classification | `semver-major/minor/patch`, dependency type, `maintainer-changes`, CVSS | empty type on digest and SHA bumps |
| dependency-review-action | safety | vulnerabilities (default: low or above, runtime scope), licenses, denied packages; Scorecard only warns below 3 | no Docker images; Actions checked by version tags only, not SHA pins; private repositories need GitHub Advanced Security |
| OSV-Scanner v2 | safety | `go.mod`, binaries, SBOMs, container images; Go call analysis on by default (`called: true/false`) | whether uncalled vulnerabilities fail the exit code is unverified |
| govulncheck | safety | reports only vulnerabilities the code can reach | JSON and SARIF output always exit 0, so the gate must parse it |
| OpenSSF Scorecard | context for new dependencies | 20 checks, 0–10 each | heuristic; advisory only |

### History, sensitivity, behavior

| Source | Serves | Key facts | Fit |
|---|---|---|---|
| code-maat (GPL-3.0) | co-change, hotspot | `coupling` (degree 0–100), `revisions`, churn, age; CSV | good offline; superseded by commercial CodeScene |
| PyDriller | churn, glue code | commits, modified files, process metrics, SZZ (finds bug-*inducing* commits, not reverts) | good glue |
| git revert convention | revert in 90 days | `This reverts commit <sha>.` | misses manual or edited reverts |
| fullsend ADR 0089 | history ≥ 3, sensitivity | score = 0.5 metadata + 0.3 git history + 0.2 issue; history part run by an LLM; security list = directory names | reuse the path list only |
| flagd / OpenFeature / Kubernetes feature list | flag default flips | declarative defaults can be diffed | good for declarative flags |
| Go pflag/env defaults, LaunchDarkly/Unleash | flag flips | no tool; SaaS flips never appear in the PR | gap |

### Multi-repo orchestration

| Tool | Cross-repo dependency | Tests together | Order / atomic | Computes reach |
|---|---|---|---|---|
| Zuul | `Depends-On:` (PR body on GitHub) | yes, speculatively | orders merges; cycles opt-in, not atomic | no |
| Mergify | `Depends-On:` (`owner/repo#n` or a URL) | not across repositories (unverified) | declare-and-wait; a cycle blocks forever | no (uses scopes for batching only) |
| Gerrit topics | whole-topic submit | via Zuul | atomic within one repository only | no |
| Prow Tide | no | batches within one repository | one pool per repository and branch | no |
| GitHub merge queue | no | groups within one repository | per repository and branch | no |
| Kodiak, Renovate automerge | no | no | label- or check-driven | no |
| GitHub rulesets | no | no | can require checks, reviews, a merge queue, reviewers by path | no |

## Suggested doc changes (not applied)

- **Sensitivity row:** "security paths (auth, secrets, RBAC, crypto, tokens, policies), CI or migrations → T3", tools
  "ADR 0089's path list". Drop "a risky new import → T2", which has no tool.
- **History row:** drop "ADR 0089 score ≥ 3" (not reproducible). Tools: `git log` revert parsing, code-maat.
- **Interface compatibility row:** tools "crdify (CRDs), go-apidiff (Go APIs), oasdiff (REST), `buf breaking`
  (gRPC)". Add: changes to CEL rules or webhooks escalate.
- **Dependencies row:** tools "Renovate or Dependabot update type checked against the `go.mod` diff;
  dependency-review, govulncheck or OSV-Scanner". Add: a bump the bot can't classify is not T0.
- **Question 6:** add the lean from bottom line 8.
- **Alternatives:** "Merge-on-green tools (Renovate automerge, Kodiak) and merge engines (Mergify, Prow Tide, Zuul,
  GitHub rulesets) decide from PR attributes, changed paths, declared `Depends-On` links and a check list; none
  computes reach."

## Sources

Interface: go-apidiff, x/exp/apidiff, gorelease, crdify (README, releases, validations, configuration),
crd-schema-checker, kube-api-linter, oasdiff (breaking-changes doc and three checks), buf breaking (overview, rules,
CLI), japicmp. Dependencies: Renovate configuration options and `options/index.ts`, versioning, automerge,
minimum release age, templates, Go docs; Dependabot fetch-metadata and GitHub Dependabot docs; dependency-review-action;
dependency-graph ecosystems; OSV-Scanner docs; govulncheck; OpenSSF Scorecard. History: fullsend ADR 0089 and the
agents repository's `pr-risk-assessment` skill and script; code-maat; PyDriller; git-revert; GitHub revert docs;
CodeScene; flagd; OpenFeature spec and CLI; Kubernetes feature gates, `versioned_feature_list.yaml`,
`verify-featuregates.sh`; LaunchDarkly code references; Unleash. Orchestration: Zuul gating, GitHub driver and queues;
Gerrit configuration and cross-repository changes; Prow Tide; GitHub merge queue, rulesets and stacked PRs; Mergify
merge queue, merge action, built-in protections and scopes; Kodiak; Renovate automerge; bors-ng.

Caveat: page reads went through a summarizer, so spot-check quoted commands against the source before relying on them.
