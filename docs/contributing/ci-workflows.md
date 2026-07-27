# CI Workflow Conventions

Guidelines for writing and maintaining GitHub Actions workflows in this repository.

## Concurrency groups

Use `${{ github.workflow }}` as the workflow identifier in concurrency groups -- never duplicate the workflow name as a hardcoded string prefix. Pair it with a run-specific key (PR number, branch ref, issue number) so each logical unit of work gets its own group:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}
  cancel-in-progress: true
```

**Exception: reusable workflows.** Reusable workflows (`on: workflow_call`) must use a hardcoded role-specific prefix instead of `${{ github.workflow }}`, because `github.workflow` resolves to the *caller's* workflow name in `workflow_call` context, not the reusable workflow's own name. Using `${{ github.workflow }}` in a reusable workflow would produce incorrect concurrency scoping -- potentially canceling the caller's own run rather than a previous run of the same reusable workflow.

The repo's caller workflows (e.g., `code.yml` using `fullsend-code-`, `review.yml` using `fullsend-review-`) follow this pattern:

```yaml
# Caller workflow (workflow_dispatch) -- hardcoded prefix, not github.workflow
concurrency:
  group: fullsend-code-${{ inputs.source_repo }}-${{ fromJSON(inputs.event_payload).issue.number }}
  cancel-in-progress: true
```

### `cancel-in-progress` for `pull_request_target`

When using `pull_request_target`, gate `cancel-in-progress` carefully. Canceling in-progress runs on the default branch can interrupt deployments or other side-effecting jobs. The `e2e.yml` workflow shows the established pattern -- cancel for PRs but not for pushes to `main`:

```yaml
concurrency:
  group: >-
    ${{ github.event_name == 'pull_request_target'
        && format('e2e-{0}', github.event.pull_request.number)
        || format('{0}-{1}', github.workflow, github.ref) }}
  cancel-in-progress: >-
    ${{ github.event_name == 'pull_request_target'
        || github.ref != 'refs/heads/main' }}
```
