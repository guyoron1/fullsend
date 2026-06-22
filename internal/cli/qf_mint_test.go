package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GH-73-TC-026: Verify add-role with slug and PEM file
func TestQF_ParseMintAddRoleMode_SlugPEM(t *testing.T) {
	mode, err := parseMintAddRoleMode("test-agent", "/tmp/key.pem", "", false)
	require.NoError(t, err)
	assert.Equal(t, addRoleModeSlugPEM, mode, "should select slug+PEM mode")
}

// GH-73-TC-027: Verify add-role with existing PEM secret
func TestQF_ParseMintAddRoleMode_ExistingSecret(t *testing.T) {
	mode, err := parseMintAddRoleMode("test-agent", "", "", true)
	require.NoError(t, err)
	assert.Equal(t, addRoleModeExistingSecret, mode, "should select existing-secret mode")
}

// GH-73-TC-028: Verify error for missing project flag
// Note: The --project flag is validated at the cobra command level (MarkFlagRequired),
// not in parseMintAddRoleMode. We test parseMintAddRoleMode with no valid mode selected.
func TestQF_ParseMintAddRoleMode_NoInputMode(t *testing.T) {
	_, err := parseMintAddRoleMode("", "", "", false)
	assert.Error(t, err, "should error when no input mode is specified")
}

// GH-73-TC-029: Verify mutual exclusivity of input modes
func TestQF_ParseMintAddRoleMode_MutuallyExclusive(t *testing.T) {
	_, err := parseMintAddRoleMode("test-agent", "/tmp/key.pem", "", true)
	assert.Error(t, err, "should error when both --pem-file and --existing-secret are provided")
}

// GH-73-TC-026 supplemental: Verify browser mode with org
func TestQF_ParseMintAddRoleMode_BrowserMode(t *testing.T) {
	mode, err := parseMintAddRoleMode("", "", "my-org", false)
	require.NoError(t, err)
	assert.Equal(t, addRoleModeBrowser, mode, "should select browser mode when org is specified")
}

// GH-73-TC-029 supplemental: Verify org cannot be combined with slug flags
func TestQF_ParseMintAddRoleMode_OrgWithSlug(t *testing.T) {
	_, err := parseMintAddRoleMode("test-agent", "", "my-org", false)
	assert.Error(t, err, "should error when --org combined with --slug")
}
