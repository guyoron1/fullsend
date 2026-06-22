package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Cross-File Verification — Scaffold Embed & Deployment Tests

STP Reference: outputs/stp/GH-70/GH-70_test_plan.md
STD Reference: outputs/std/GH-70/GH-70_test_description.yaml
Jira: GH-1835

Tests verify that cross-file verification skill files are properly
embedded in the scaffold binary and deployable to enrolled repositories.
Note: skills/ is a layered directory, so WalkFullsendRepoAll is used
instead of WalkFullsendRepo (which skips layered dirs by design).
*/

func TestQFCrossFileVerificationScaffoldEmbed(t *testing.T) {
	t.Run("[test_id:TS-GH-1835-006] should include SKILL.md in embedded filesystem", func(t *testing.T) {
		/*
			Preconditions:
			    - scaffold embed.FS includes skills directory

			Steps:
			    1. Initialize path collector slice
			    2. Walk embedded filesystem via WalkFullsendRepoAll
			       (skills/ is a layered dir, skipped by WalkFullsendRepo)
			    3. Collect all walked file paths
			    4. Search collected paths for skills/code-review/SKILL.md

			Expected:
			    - Walk completes without error
			    - Collected paths contain "skills/code-review/SKILL.md"
		*/
		var found []string
		err := WalkFullsendRepoAll(func(path string, content []byte) error {
			found = append(found, path)
			return nil
		})
		require.NoError(t, err, "WalkFullsendRepoAll must complete without error")
		assert.Contains(t, found, "skills/code-review/SKILL.md",
			"SKILL.md must be discoverable in the embedded filesystem")
	})

	t.Run("[test_id:TS-GH-1835-007] should include correctness.md in scaffold embed", func(t *testing.T) {
		/*
			Preconditions:
			    - scaffold embed.FS includes sub-agents directory

			Steps:
			    1. Read correctness.md from scaffold embed
			    2. Verify no error returned
			    3. Verify content is non-empty

			Expected:
			    - FullsendRepoFile returns content without error
			    - Returned content has length > 0
		*/
		content, err := FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err, "correctness.md must be readable from scaffold embed")
		assert.NotEmpty(t, content, "correctness.md must have non-empty content")
	})

	t.Run("[test_id:TS-GH-1835-008] should deploy updated files via layered content", func(t *testing.T) {
		/*
			Preconditions:
			    - scaffold embed.FS includes layered content

			Steps:
			    1. Walk layered content via WalkLayeredContent
			    2. Collect files matching SKILL.md and correctness.md paths
			    3. Verify both files are present in layered content
			    4. Verify both files contain cross-file verification instructions

			Expected:
			    - WalkLayeredContent includes SKILL.md
			    - WalkLayeredContent includes correctness.md
			    - Both files contain "Cross-file verification" text

			Note: skills/ is a layered directory — files are delivered at
			runtime via reusable workflows, not via CollectInstallFiles.
			WalkLayeredContent is the correct function for verifying
			deployment of layered content.
		*/
		collected := make(map[string]string)
		err := WalkLayeredContent(func(path string, content []byte) error {
			if path == "skills/code-review/SKILL.md" ||
				path == "skills/pr-review/sub-agents/correctness.md" {
				collected[path] = string(content)
			}
			return nil
		})
		require.NoError(t, err, "WalkLayeredContent must complete without error")

		skillContent, ok := collected["skills/code-review/SKILL.md"]
		require.True(t, ok, "SKILL.md must be present in layered content")
		assert.True(t, strings.Contains(skillContent, "Cross-file verification"),
			"deployed SKILL.md must contain cross-file verification instructions")

		correctnessContent, ok := collected["skills/pr-review/sub-agents/correctness.md"]
		require.True(t, ok, "correctness.md must be present in layered content")
		assert.True(t, strings.Contains(correctnessContent, "Cross-file verification"),
			"deployed correctness.md must contain cross-file verification instructions")
	})
}
