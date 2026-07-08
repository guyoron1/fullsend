# Configuring GitHub For Fullsend

The goal of this document is that you configure Fullsend for your GitHub repository.

## Prerequisites

* You have your WIF provider URL from [Getting Inference](getting-inference.md).
* Download the latest [fullsend](https://github.com/fullsend-ai/fullsend/releases) CLI.
* Download the latest [gh](https://cli.github.com/) CLI and authenticate with it.

### Token resolution

The fullsend CLI resolves a GitHub token in this order:

1. `GH_TOKEN` environment variable
2. `GITHUB_TOKEN` environment variable
3. `gh auth token` (output of the GitHub CLI)

For most organizations, running `gh auth login` is sufficient.

### Organizations that restrict classic PATs

If your organization restricts personal access token types (Settings >
Personal access tokens > restrict access via PAT type), the default
token from `gh auth login` may be rejected by the GitHub API. In that
case, create a fine-grained personal access token manually at
<https://github.com/settings/personal-access-tokens/new> and grant it
the following repository permissions:

| Permission | Level | Why |
|---|---|---|
| Contents | Read and write | Commits `.fullsend/config.yaml` and scaffold files |
| Workflows | Read and write | Writes/updates files under `.github/workflows/` |
| Secrets | Read and write | Sets `FULLSEND_GCP_PROJECT_ID` / `FULLSEND_GCP_WIF_PROVIDER` |
| Variables | Read and write | Sets `FULLSEND_MINT_URL` / `FULLSEND_GCP_REGION` |
| Metadata | Read-only | GitHub-required baseline (granted automatically) |
| Pull requests | Read and write | Only needed without `--direct` |

Export the token before running fullsend so it takes priority:

```bash
export GH_TOKEN="github_pat_..."
fullsend github setup <org>/<repo> ...
```

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
