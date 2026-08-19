# Cross-Repo Work Decomposition

One change, several repositories. How is a single unit of intent split across repos, who owns the parts, and what makes the parts one thing rather than several unrelated issues?

**Related:**
- [agent-architecture.md](agent-architecture.md) — the repo as coordinator
- [codebase-context.md](codebase-context.md) — context loading for multi-repo changes
- [code-review.md](code-review.md) — reviewing changes that span repos
- [intent-representation.md](intent-representation.md) — what authorizes the work
- [flapping-convergence.md](flapping-convergence.md) — what happens when coordinated parts do not converge

## The problem

An organization's change does not always fit in one repository. An API field is added in a schema repo, consumed in two services, surfaced in a client, and documented in a docs repo. Today fullsend handles the parts; nothing handles the whole.

### What exists today

Two primitives already touch this space, and they are the right place to start.

**Prerequisite issue creation.** The triage agent can file blocking issues in other repositories when it identifies an upstream dependency, governed by the `create_issues.allow_targets` allowlist in `config.yaml`. When the allowlist is empty the agent still identifies the dependency and drafts an issue body for a human to file. This produces a dependency edge: *this issue is blocked by that one, over there*.

**Independent per-repo discovery.** Each enrolled repo polls its intent source on its own schedule with its own filter. From [jira-integration.md](../guides/user/jira-integration.md): *"Each repo runs its own poll workflow and gets an isolated view of coordination state… There is no shared state between repos."* A Jira issue relevant to three repos is discovered three times, independently, and may dispatch three agent runs that know nothing about each other.

Both behaviors are deliberate and consistent with [the repo as coordinator](agent-architecture.md#interaction-model-the-repo-as-coordinator). Neither one decomposes work.

### What is missing

Four capabilities, in rough order of difficulty:

**Decomposition.** Turning one statement of intent into repo-scoped units of work, where the split is driven by what each repo actually contains rather than by a guess. This is the entry point: without it, either a human does the split by hand, or each repo's agent independently interprets the whole intent and implements its own idea of the part that belongs to it.

**Ownership.** Deciding which repo owns which part, and recording that decision somewhere both humans and agents can read. The `blocked` label expresses *ordering* between issues; it does not express *membership* in a shared unit of work.

**Aggregation.** Knowing whether the whole change is done. Today three repo-scoped PRs are three PRs. Nothing reports "two of three merged, the client is still open," and nothing prevents an intent from being half-shipped indefinitely.

**Ordering and atomicity.** Some cross-repo changes are a chain — the schema must merge before the consumers compile. Some are atomic — merging one half without the other breaks production. Agents currently have no way to express either constraint, so a correct sequencing depends on humans noticing.

### Why this matters at scale

For one team with three repos, a human holds the whole change in their head and the missing coordination is invisible. The cost appears with scale and with distance: an org with forty enrolled repos, where the person who filed the intent does not know which services consume the field they are changing.

The failure mode is not a bad PR — each individual PR may be perfectly good. The failure mode is a **partially applied change**: the schema merges, the consumers do not, and nothing in the system knows the difference between "in progress" and "abandoned halfway."

## Shapes of cross-repo work

These do not all need the same solution, and conflating them is a good way to over-build.

| Shape | Example | What it needs |
|---|---|---|
| **Prerequisite chain** | Fix an upstream bug, then use the fix | Ordering. Largely handled today by `blocked` + prerequisite issue creation |
| **Parallel slices** | Same feature implemented independently in three services | Decomposition and aggregation; ordering does not matter |
| **Atomic contract change** | Schema change plus every consumer | Decomposition, ordering, and a merge policy that prevents half-application |
| **Mechanical fan-out** | Bump a dependency across forty repos | Little decomposition, heavy aggregation and reporting |

Mechanical fan-out is the shape most orgs meet first and the one most easily solved without a general model — a script that files N identical issues gets most of the value. The hard shape is the atomic contract change.

## Approaches

### A. Do nothing beyond what exists

Humans decompose; agents execute per-repo parts as ordinary issues. Prerequisite creation and `blocked` cover the chain case.

- Preserves the coordination model exactly. Zero new state.
- Puts the whole burden on the human who files the intent, and the burden grows with the number of repos they must know about.
- Partially applied changes remain invisible to the system.

### B. Decomposition as an agent output, coordination in the forge

An agent reads the intent, inspects the candidate repos, and emits repo-scoped issues linked to a parent issue. Membership lives in the forge — a parent issue with a task list, or an issue-link type — rather than in a fullsend-owned store. Existing agents then work the children as ordinary issues.

- Reuses the coordination substrate already in use: issues, links, labels.
- Aggregation becomes readable by humans and agents alike from the parent.
- Needs a decision about where the parent lives when no repo is the natural home, and forges differ in their linking primitives (GitHub task lists and sub-issues, GitLab epics, Jira issue links).
- Does not by itself solve ordering or atomicity.

### C. A cross-repo work object owned by fullsend

A first-class record — a unit of work with member repos, per-repo status, ordering constraints, and completion criteria — stored outside the forge.

- Expresses everything, including atomicity and merge ordering.
- Introduces exactly the shared state the architecture has avoided so far, plus a component whose availability every enrolled repo now depends on.
- Should be adopted only if B provably cannot express the atomic contract change.

### D. Worked example: how Forge does it

[Forge](https://github.com/forge-sdlc/forge), an independent Jira-driven SDLC orchestrator, implements decomposition, and its approach is worth reading precisely because its trade-offs are visible.

A ticket is decomposed against the repositories configured for its project: the decomposition agent inspects each target repo before writing anything, and its prompt requires that *every file path in a plan must be a real path discovered during inspection, or a clearly marked new path in an existing directory* — with an explicit instruction to return a blocking note rather than guess when repo inspection fails. It caps sprawl deliberately (one epic for a simple change, three to five for a large one, "fewer is better," no artificial config/validation/tests splits). The resulting repo-scoped units are implemented and reviewed independently, with the parent ticket tracking the multi-repo PR lifecycle.

Two lessons transfer regardless of which approach is chosen. **Repository grounding is what separates a decomposition from a guess** — a split proposed without reading the repos will invent paths and miss the consumer nobody remembered. And **decomposition needs an explicit sizing discipline**, because an agent asked to split work will happily split it further than anyone wants.

The cost is equally visible: Forge holds workflow state in a central service, which is the coordination model this project has deliberately not adopted.

## Tension with the repo as coordinator

This is the crux, and it deserves stating plainly rather than being resolved by assumption.

"The repo is the coordinator" has held because every unit of work has so far fit inside one repo. A cross-repo change is precisely the case where no single repo can be the coordinator, because no single repo can see the whole change. Approach B keeps the principle by making the *parent issue* the coordinator; approach C abandons it.

If B is sufficient, the principle survives intact and cross-repo work becomes a naming and linking convention rather than an architecture change. Establishing whether B is sufficient — specifically for the atomic contract change — is the most useful next step this document can point at.

## Relationship to other problem areas

- **[Codebase Context](codebase-context.md)** — decomposition requires reading several repos at once, which is the multi-repo context loading question already raised there.
- **[Code Review](code-review.md)** — asks how a reviewer evaluates a change spanning repos. Decomposition determines what the reviewer of each part can see of the whole.
- **[Intent Representation](intent-representation.md)** — if one intent authorizes work in five repos, each repo needs to verify that authorization locally.
- **[Autonomy Spectrum](autonomy-spectrum.md)** — autonomy is currently per repo. A change spanning an autonomous repo and a gated one takes which policy?
- **[Flapping and Convergence](flapping-convergence.md)** — coordinated parts can oscillate against each other; a partially applied change is a non-converged state that no per-repo circuit breaker detects.

## Open questions

- Which shape should be solved first? Mechanical fan-out is the most common and the least architecturally interesting; the atomic contract change is the reverse.
- Where does the parent work item live when no repo is the natural owner — an org-level tracking repo, the intent source, or `.fullsend` itself?
- Can decomposition be expressed as an ordinary agent with a structured output, so that it needs no new execution machinery?
- Should an agent ever be permitted to decide that a change is atomic, or is "these must land together" always a human assertion?
- How is a partially applied change detected and reported, and to whom?
- Does cross-repo decomposition require cross-repo credentials, or can each part be executed under the target repo's own existing identity? The latter preserves [credential isolation](../ADRs/0017-credential-isolation-for-sandboxed-agents.md); the former does not.
