package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Scaffold Embedding Tests — GH-2096

Validates that the security-triage sub-agent definition is correctly embedded
in the scaffold and included in install file collection.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-019, TS-GH-2096-020, TS-GH-2096-021
*/

const securityTriageScaffoldPath = "skills/pr-review/sub-agents/security-triage.md"

func TestScaffoldEmbedding(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
			- go:embed directive includes sub-agents/security-triage.md
	*/

	// TS-GH-2096-019: Verify FullsendRepoFile reads security-triage.md
	t.Run("FullsendRepoFile reads security-triage.md", func(t *testing.T) {
		content, err := scaffold.FullsendRepoFile(securityTriageScaffoldPath)
		require.NoError(t, err,
			"FullsendRepoFile must not return error for security-triage.md")
		require.NotEmpty(t, content,
			"FullsendRepoFile must return non-empty content")

		// Content should be valid markdown (starts with frontmatter or heading)
		contentStr := string(content)
		assert.True(t,
			contentStr[0] == '#' || contentStr[:3] == "---",
			"security-triage.md content must be valid markdown (starts with # or ---)")
	})

	// TS-GH-2096-020: Verify CollectInstallFiles includes security-triage.md
	t.Run("scaffold walk includes security-triage.md", func(t *testing.T) {
		// WalkFullsendRepoAll includes layered directories (skills/ is layered)
		var found bool
		err := scaffold.WalkFullsendRepoAll(func(path string, content []byte) error {
			if path == securityTriageScaffoldPath {
				found = true
			}
			return nil
		})
		require.NoError(t, err, "WalkFullsendRepoAll must not error")
		assert.True(t, found,
			"security-triage.md must be present in scaffold walk output at %q",
			securityTriageScaffoldPath)
	})

	// TS-GH-2096-021: Verify installed file content matches embedded source
	t.Run("installed file content matches embedded source", func(t *testing.T) {
		// Read embedded source
		embeddedContent, err := scaffold.FullsendRepoFile(securityTriageScaffoldPath)
		require.NoError(t, err)
		require.NotEmpty(t, embeddedContent)

		// Walk scaffold and find the same file
		var walkedContent []byte
		err = scaffold.WalkFullsendRepoAll(func(path string, content []byte) error {
			if path == securityTriageScaffoldPath {
				walkedContent = make([]byte, len(content))
				copy(walkedContent, content)
			}
			return nil
		})
		require.NoError(t, err)
		require.NotEmpty(t, walkedContent,
			"security-triage.md must be found during scaffold walk")

		assert.Equal(t, embeddedContent, walkedContent,
			"embedded content must match walked content byte-for-byte")
	})
}
