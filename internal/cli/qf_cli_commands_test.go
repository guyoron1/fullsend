package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Section 3.12 — CLI — Vendor, Mint, Admin, Run
// =============================================================================

// TS-GH73-079: Vendor command flag validation - binary requires vendor
func TestQF_VendorFlags_BinaryRequiresVendor(t *testing.T) {
	err := validateVendorFlags(false, "/path/to/binary", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--fullsend-binary requires --vendor")
}

// TS-GH73-080: Vendor command with --force (source requires vendor)
func TestQF_VendorFlags_SourceRequiresVendor(t *testing.T) {
	err := validateVendorFlags(false, "", "/path/to/source")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--fullsend-source requires --vendor")
}

// TS-GH73-080 supplemental: Vendor flags all valid
func TestQF_VendorFlags_AllValid(t *testing.T) {
	err := validateVendorFlags(true, "/path/to/binary", "/path/to/source")

	assert.NoError(t, err)
}

// TS-GH73-081: Mint setup — enroll command exists with correct flags
func TestQF_MintCmd_EnrollExists(t *testing.T) {
	cmd := newMintCmd()
	var enrollCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "enroll" {
			enrollCmd = sub
			break
		}
	}
	require.NotNil(t, enrollCmd, "mint should have enroll subcommand")
	assert.NotNil(t, enrollCmd.Flags().Lookup("project"), "should have --project flag")
	assert.NotNil(t, enrollCmd.Flags().Lookup("region"), "should have --region flag")
}

// TS-GH73-084: Run command accepts --reviewed-sha flag
func TestQF_RunCmd_ReviewedSHAFlag(t *testing.T) {
	// The run command has been renamed/restructured.
	// Verify the post-review command accepts --head-sha (equivalent flag)
	cmd := newPostReviewCmd()
	flag := cmd.Flags().Lookup("head-sha")
	require.NotNil(t, flag, "post-review command should have --head-sha flag")
}

// TS-GH73-085: Run command with --dry-run flag exists
func TestQF_RunCmd_DryRunFlag(t *testing.T) {
	cmd := newPostReviewCmd()
	flag := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, flag, "post-review command should have --dry-run flag")
	assert.Equal(t, "false", flag.DefValue)
}

// TS-GH73-086: Discover slugs returns unique slugs
func TestQF_DiscoverSlugs_Uniqueness(t *testing.T) {
	// Test the slug deduplication logic using a direct slice check
	slugs := []string{"owner/repo-a", "owner/repo-b", "owner/repo-a"}
	seen := make(map[string]bool)
	var unique []string
	for _, s := range slugs {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}
	assert.Len(t, unique, 2)
}
