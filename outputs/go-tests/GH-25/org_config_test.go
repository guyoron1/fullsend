//go:build e2e

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/config"
)

/*
OrgConfig CreateIssues & MintURL Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

func TestOrgConfigCreateIssues(t *testing.T) {
	// [test_id:TS-GH-25-046] should parse create_issues allow_targets correctly
	t.Run("[test_id:TS-GH-25-046] should parse create_issues allow_targets correctly", func(t *testing.T) {
		yamlData := []byte(`
version: "2"
dispatch:
  platform: github
agents:
  - role: triage
    name: triage
    slug: triage-agent
repos:
  myrepo:
    enabled: true
create_issues:
  allow_targets:
    orgs:
      - "upstream-org"
      - "partner-org"
    repos:
      - "upstream-org/shared-lib"
      - "partner-org/api"
`)
		cfg, err := config.ParseOrgConfig(yamlData)

		require.NoError(t, err)
		require.NotNil(t, cfg.CreateIssues, "CreateIssues should be parsed")
		assert.Equal(t, []string{"upstream-org", "partner-org"}, cfg.CreateIssues.AllowTargets.Orgs)
		assert.Equal(t, []string{"upstream-org/shared-lib", "partner-org/api"}, cfg.CreateIssues.AllowTargets.Repos)
	})

	// [test_id:TS-GH-25-047] should use empty defaults without create_issues section
	t.Run("[test_id:TS-GH-25-047] should use empty defaults without create_issues section", func(t *testing.T) {
		yamlData := []byte(`
version: "2"
dispatch:
  platform: github
agents:
  - role: triage
    name: triage
    slug: triage-agent
repos:
  myrepo:
    enabled: true
`)
		cfg, err := config.ParseOrgConfig(yamlData)

		require.NoError(t, err)
		assert.Nil(t, cfg.CreateIssues,
			"CreateIssues should be nil when not present in YAML (pointer field)")
	})
}

func TestOrgConfigMintURL(t *testing.T) {
	// [test_id:TS-GH-25-048] should parse MintURL from dispatch.mint_url
	t.Run("[test_id:TS-GH-25-048] should parse MintURL from dispatch.mint_url", func(t *testing.T) {
		yamlData := []byte(`
version: "2"
dispatch:
  platform: github
  mode: oidc-mint
  mint_url: https://mint.example.com/api/v1/token
agents:
  - role: triage
    name: triage
    slug: triage-agent
repos:
  myrepo:
    enabled: true
`)
		cfg, err := config.ParseOrgConfig(yamlData)

		require.NoError(t, err)
		assert.Equal(t, "https://mint.example.com/api/v1/token", cfg.Dispatch.MintURL,
			"MintURL should be parsed from dispatch.mint_url")
		assert.Equal(t, "oidc-mint", cfg.Dispatch.Mode,
			"Mode should be parsed alongside MintURL")
	})
}
