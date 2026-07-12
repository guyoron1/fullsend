# Configuring GitHub For Fullsend

The goal of this document is that you configure Fullsend for your GitHub repository.

## Prerequisites

* You have your WIF provider URL from [Getting Inference](getting-inference.md).
* Download the latest [fullsend](https://github.com/fullsend-ai/fullsend/releases) CLI.
* Download the latest [gh](https://cli.github.com/) CLI and authenticate with it.

### Token resolution

The fullsend CLI resolves GitHub credentials in the following order:

1. **`GH_TOKEN`** environment variable
2. **`GITHUB_TOKEN`** environment variable
3. **`gh auth token`** (from `gh auth login`)

The first non-empty value wins. If your organization restricts token types
(see below), export a fine-grained PAT as `GH_TOKEN` to override the default
`gh` credential.

### Organizations that restrict classic PATs

Some GitHub organizations enforce a policy that blocks classic personal access
tokens. When this policy is active, the default token from `gh auth login`
(which is a classic PAT) will be rejected with a 403 error.

To work with these organizations, create a **fine-grained personal access
token** and export it as `GH_TOKEN`:

1. Go to <https://github.com/settings/personal-access-tokens/new>
2. Scope the token to your target organization
3. Grant the following **repository permissions**:

| Permission | Level | Why |
|---|---|---|
| Contents | Read and write | Commits `.fullsend/config.yaml` and scaffold files |
| Workflows | Read and write | Writes/updates files under `.github/workflows/` |
| Secrets | Read and write | Sets `FULLSEND_GCP_PROJECT_ID` / `FULLSEND_GCP_WIF_PROVIDER` |
| Variables | Read and write | Sets `FULLSEND_MINT_URL` / `FULLSEND_GCP_REGION` |
| Metadata | Read-only | GitHub-required baseline (automatically selected) |
| Pull requests | Read and write | Only needed without `--direct` |

4. Export the token: `export GH_TOKEN='github_pat_...'`

## Installing GitHub Applications

In order to use Fullsend install the following applications to your organization
and provide them permissions to the repository you want to install Fullsend to.

| Role | Installation URL |
|------|-----------------|
| fullsend | <https://github.com/apps/fullsend-ai-fullsend/installations/new> |
| triage | <https://github.com/apps/fullsend-ai-triage/installations/new> |
| coder | <https://github.com/apps/fullsend-ai-coder/installations/new> |
| review | <https://github.com/apps/fullsend-ai-review/installations/new> |
| retro | <https://github.com/apps/fullsend-ai-retro/installations/new> |
| prioritize | <https://github.com/apps/fullsend-ai-prioritize/installations/new> |

## Configuring GitHub

Run the command:

```bash
fullsend github setup <org>/<repo> \
  --inference-project "<gcp-project>" \
  --inference-wif-provider "<wif-provider-url>"
```

Where `<org>/<repo>` refers to the GitHub organization and repository you want to enable inference
for, `<gcp-project>` is your GCP project name, and `<wif-provider-url>` is the WIF Provider URL
created at [Getting Inference](getting-inference.md).

The command creates files, secrets and variables in your repository.

## Testing Fullsend

After installing open a new issue or comment `/fs-triage` in an open issue. Then visit the
Actions tab to see the Fullsend workflow in action. In some minutes the
`fullsend-ai-triage` bot should post a comment in the issue.

## Next steps

* Read [Organization installation mode](org-mode.md) to learn how to share GCP project with other repositories
within your GitHub organization.
* Read the [Default Agents](../../agents/README.md) section to learn about the default agents Fullsend
ships with.
* Explore other sections of this documentation for more information.
