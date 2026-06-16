//go:build e2e

package tests

/*
Model Provider Definition Loading Tests — GH-18

STP Reference: outputs/stp/GH-18/GH-18_test_plan.md
Jira: GH-18
Requirement: Model provider definitions load correctly
*/

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
)

// TestMultipleProvidersLoadedFromDirectory verifies that LoadProviderDefs
// reads all YAML files from a directory and returns them as ProviderDef structs.
//
// [test_id:TS-GH-18-028]
func TestMultipleProvidersLoadedFromDirectory(t *testing.T) {
	// Setup: Create temp directory with multiple provider YAML files
	tmpDir := t.TempDir()

	provider1 := `name: "openai"
type: "api"
credentials:
  api_key: "${OPENAI_API_KEY}"
`
	provider2 := `name: "anthropic"
type: "api"
credentials:
  api_key: "${ANTHROPIC_API_KEY}"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openai.yaml"), []byte(provider1), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "anthropic.yaml"), []byte(provider2), 0o644))

	// Execute: Load provider definitions
	defs, err := harness.LoadProviderDefs(tmpDir)

	// Assert: All providers loaded
	require.NoError(t, err, "LoadProviderDefs should not return error")
	assert.Len(t, defs, 2, "Should load 2 provider definitions")

	// Verify provider names are present
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	assert.True(t, names["openai"], "Should contain openai provider")
	assert.True(t, names["anthropic"], "Should contain anthropic provider")
}

// TestCredentialsMappedCorrectlyPerProvider verifies that each loaded ProviderDef
// has its credential configuration correctly parsed from the YAML.
//
// [test_id:TS-GH-18-029]
func TestCredentialsMappedCorrectlyPerProvider(t *testing.T) {
	// Setup: Create provider YAML with credential mapping
	tmpDir := t.TempDir()

	providerYAML := `name: "openai"
type: "api"
credentials:
  api_key: "${OPENAI_API_KEY}"
  org_id: "${OPENAI_ORG_ID}"
config:
  base_url: "https://api.openai.com/v1"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "openai.yaml"), []byte(providerYAML), 0o644))

	// Execute: Load and inspect credentials
	defs, err := harness.LoadProviderDefs(tmpDir)
	require.NoError(t, err)
	require.Len(t, defs, 1)

	provider := defs[0]

	// Assert: Credential env vars mapped correctly
	assert.Equal(t, "openai", provider.Name, "Provider name should match")
	assert.Equal(t, "api", provider.Type, "Provider type should match")
	assert.Equal(t, "${OPENAI_API_KEY}", provider.Credentials["api_key"],
		"api_key credential should be mapped")
	assert.Equal(t, "${OPENAI_ORG_ID}", provider.Credentials["org_id"],
		"org_id credential should be mapped")

	// Verify config section is also parsed
	assert.Equal(t, "https://api.openai.com/v1", provider.Config["base_url"],
		"Config base_url should be parsed")
}

// TestErrorForMissingRequiredNameField verifies that LoadProviderDefs returns
// an error when a provider YAML file is missing the required "name" field.
//
// [test_id:TS-GH-18-030]
func TestErrorForMissingRequiredNameField(t *testing.T) {
	// Setup: Create provider YAML without name field
	tmpDir := t.TempDir()

	invalidYAML := `type: "api"
credentials:
  api_key: "${API_KEY}"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0o644))

	// Execute: Attempt to load provider defs
	defs, err := harness.LoadProviderDefs(tmpDir)

	// Assert: Error returned referencing missing name
	assert.Error(t, err, "LoadProviderDefs should return error for missing name")
	assert.Nil(t, defs, "Defs should be nil on error")
	assert.Contains(t, err.Error(), "name",
		"Error message should reference the missing name field")
}

// TestErrorForMissingRequiredTypeField verifies that LoadProviderDefs returns
// an error when a provider YAML file is missing the required "type" field.
//
// [test_id:TS-GH-18-031]
func TestErrorForMissingRequiredTypeField(t *testing.T) {
	// Setup: Create provider YAML without type field
	tmpDir := t.TempDir()

	invalidYAML := `name: "openai"
credentials:
  api_key: "${API_KEY}"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0o644))

	// Execute: Attempt to load provider defs
	defs, err := harness.LoadProviderDefs(tmpDir)

	// Assert: Error returned referencing missing type
	assert.Error(t, err, "LoadProviderDefs should return error for missing type")
	assert.Nil(t, defs, "Defs should be nil on error")
	assert.Contains(t, err.Error(), "type",
		"Error message should reference the missing type field")
}
