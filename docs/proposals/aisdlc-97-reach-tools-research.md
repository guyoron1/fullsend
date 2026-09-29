# Reach (blast radius) tools: what the documentation says (2026-09-29)

Scope: documentation only, nothing installed or run (evidence level `doc` in AISDLC-99's rubric). Question: which
tools can compute the Reach signal of AISDLC-97: dependents (direct and transitive, 10+ → T2), depth (3+ layers →
+1), consumers in another repository (→ T3), and confidence (low → +1), for a GitHub org with private Go
repositories such as fullsend-ai.

## Bottom lines

1. **Reach has three layers, and no single tool covers all three.** Inside one repository, across the org's
   repositories, and at service/API level (runtime callers, CRD users) need different sources.
2. **Inside one repository, it's solved.** Take the build graph as JSON, invert it, and compute count and depth
   with one small breadth-first search. Only Bazel `rdeps` gives reverse dependents and depth natively. Go's
   `go list -json` is forward-only, but exact and cheap once inverted.
3. **Across private repositories, GitHub and deps.dev don't help.** GitHub's "Dependents / Used by" covers public
   repositories only, lives in the web UI only, and its counts are approximate. deps.dev is public-only and has no
   dependents data for Go.
4. **The workable options across the org:**
   - **Org code search over `go.mod`:** zero infrastructure, direct consumers only, medium confidence.
   - **GUAC over SBOMs:** the only option with reverse dependents and hop depth. It needs good SBOMs from every
     repository.
   - **Sourcegraph with scip-go:** symbol-precise ("who calls this changed function"). About $16K a year or more.
5. **SBOM quality is the real dependency, not the index.** Every repository must publish SBOMs with real
   dependency edges. Syft's Go output is flat, so prefer cdxgen, and Konflux SBOMs where they exist. Package IDs must
   match across repositories, and artifacts must map back to repositories (SLSA provenance carries that).
6. **Nothing sees service/API-level consumers automatically.** REST and gRPC callers, CRD users and config consumers
   never appear in SBOMs. Backstage `consumesApi`/`dependsOn` covers them if teams maintain it (hand-written, one
   hop only); otherwise code search, with low confidence → +1.
7. **Deadline:** GitHub's synchronous SBOM endpoint `GET /repos/{o}/{r}/dependency-graph/sbom` shuts down on
   2026-11-13. Use `generate-report` then `fetch-report` instead.

## Recommended path (proposal, for AISDLC-96 and AISDLC-99 to confirm)

| Stage | Inside the repository | Across the org | Service / API |
|---|---|---|---|
| Observe mode (now) | build graph + BFS (`go list -json`, Bazel `rdeps`, Nx or Pants graph JSON) | org code search over `go.mod` | none; T3 only via the interface-compatibility signal |
| Later | same | GUAC over cdxgen or Konflux SBOMs, for depth across repositories | Backstage where it exists |
| Only if the benchmark shows many false T3s | callgraph or gopls as a confidence input | Sourcegraph + scip-go, symbol-precise | same |

**Confidence mapping for the "+1" rule:** a build graph and Sourcegraph's precise results are high. GUAC is high
when SBOMs have real edges and medium otherwise. Code search is medium (text match, default branch only). GitHub's
dependent counts are low; the docs call them approximate.

## Per tool

### Inside one repository

| Tool | What it gives | Depth | Limits | Fit |
|---|---|---|---|---|
| `go list -deps -json` | forward imports per package | invert and BFS yourself | one build configuration per run; files behind build tags are invisible | high at package level |
| x/tools `callgraph`, gopls call hierarchy | function-level calls | none natively | needs a whole program; misses reflection and dynamic calls | confidence input only |
| Bazel `rdeps(universe, x, depth)` | reverse dependents | native (depth bound, rank output) | `select()` is over-approximated (safe, inflated) | best native fit |
| Nx `affected` / graph JSON, Pants `dependents --transitive`, Turborepo | changed files → dependent projects | BFS yourself (Turborepo has none) | a lockfile change marks everything affected | high for count |
| Gradle, Maven, pipdeptree, `npm explain` | forward trees; reverse only through workarounds | partial | no first-class "my dependents" report | low |

### Across repositories

| Tool | Reverse dependents across the org | Private | Go | Depth | Cost and limits | Fit |
|---|---|---|---|---|---|---|
| GitHub dependency graph (Dependents) | web UI only | no | forward only | no | counts approximate | poor |
| deps.dev | counts only | no | no dependents for Go | no | public data | none |
| GitHub code search (`org:X filename:go.mod "module"`) | yes, by text match | yes, with a token | yes | direct only | 10 requests/min; default branch; 1,000 results | cheap first pass |
| GUAC (OpenSSF, Apache-2.0, v1.1.0) | yes: `guacone query patch --search-depth N` returns one frontier per hop | yes, self-hosted | through SBOMs | yes, hop count | Postgres plus collectors; server-side `path` is limited, so client-side BFS is recommended | best for depth |
| OWASP Dependency-Track | "which projects contain X" | yes, self-hosted | through SBOMs | no | no project-to-project edges; identity search ~7–14 s at 2.5k projects | partial (count only) |
| Sourcegraph + scip-go | yes, per symbol | yes, self-hosted | yes | hop by hop | Enterprise from ~$16K/year; cross-repo navigation breaks with the Bazel packages driver | most precise, costly |
| Renovate / Mend | no | – | forward | no | per-repository dashboard | none |

### SBOM sources and service level

| Source | Notes |
|---|---|
| cdxgen | CycloneDX from `go list -deps` and `go.mod`; keeps a real tree (required vs `// indirect`). Preferred. |
| Syft | Includes transitive Go modules, but relationships are flat, so depth is lost. |
| GitHub SBOM export | SPDX with relationships, zero infrastructure; no dependents; use the async report API. |
| Konflux | Already attaches SBOMs and SLSA provenance (with source repository and revision) to images; a ready feed for GUAC. |
| Backstage catalog | `dependsOn`, `consumesApi`, one hop, hand-maintained; the only service/API-level source. |
| CRD consumers | No tool. Go consumers of the typed API module show up in SBOMs; YAML, Helm and manifests need code search. |

## What this changed in the proposal

- **Applied:** the Reach row now lists org code search over `go.mod` and GUAC over SBOMs for cross-repository reach,
  replacing the GitHub dependency graph and deps.dev (public-only; no Go dependents).
- **Applied:** the org-wide dependency-graph question left the proposal as AISDLC-96 and AISDLC-99 scope.
- **Open:** AISDLC-99's capability matrix, Impact row ("consumer maps have no candidate yet") → the candidates above,
  at evidence level `doc`.

## Sources read

GitHub: dependency graph (about, exploring, supported ecosystems, configuring), org dependency insights, SBOM REST
and export, dependency submission, GraphQL dependency graph, code search (REST and about). deps.dev API v3 and
v3alpha, FAQ. Sourcegraph precise code navigation, cross-repository navigation blog, pricing, scip-go. Renovate
dashboard and self-hosted configuration. GUAC docs (overview, patch planning, GraphQL, ingesting SBOMs, components)
and releases. Dependency-Track features, impact analysis, v4.2 and v4.12 notes, issue #7223. Syft and Anchore Go
capabilities, cdxgen. Konflux SBOMs and attestations. Backstage software catalog, well-known relations, descriptor
format, catalog API. Go `cmd/go` list, x/tools callgraph (static, CHA, RTA, VTA) and go/packages, gopls navigation.
Bazel query and cquery. Nx affected, commands, ProjectGraph. Turborepo run. Pants introspection and dependents.
Gradle dependency debugging. Maven `dependency:tree`. pipdeptree. npm explain and ls.

Not confirmed (search snippets only): GUAC `collect registry`, the kusaridev Helm chart for GUAC, Dependency-Track's
`dependencyGraph` REST endpoints, Gradle `buildDependents` in current docs, the exact Nx graph JSON schema.
