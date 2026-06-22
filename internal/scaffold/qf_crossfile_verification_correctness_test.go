package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Cross-File Verification — Correctness Sub-Agent Tests

STP Reference: outputs/stp/GH-70/GH-70_test_plan.md
STD Reference: outputs/std/GH-70/GH-70_test_description.yaml
Jira: GH-1835

Tests verify that the correctness.md sub-agent definition contains
cross-file verification instructions. The correctness sub-agent is
dispatched by the pr-review orchestrator and must not produce findings
based on assumed file contents.
*/

func TestQFCrossFileVerificationCorrectnessSubAgent(t *testing.T) {
	// Read correctness.md once for all subtests.
	// qfNormalizeWS (defined in qf_crossfile_verification_skill_test.go) collapses
	// whitespace so assertions are immune to markdown line wrapping.
	raw, err := FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
	require.NoError(t, err, "correctness.md must be readable from scaffold embed")
	content := qfNormalizeWS(string(raw))

	t.Run("[test_id:TS-GH-1835-004] should contain cross-file verification section", func(t *testing.T) {
		/*
			Preconditions:
			    - correctness.md readable via FullsendRepoFile

			Steps:
			    1. Read correctness.md via FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
			    2. Search content for cross-file verification section heading
			    3. Search content for MUST read mandate
			    4. Search content for prohibition against probable reasoning

			Expected:
			    - Content contains "Cross-file verification" section heading
			    - Content contains "MUST read that file before asserting" mandate
			    - Content contains prohibition against reasoning about probable file contents
		*/
		assert.Contains(t, content, "Cross-file verification",
			"correctness.md must contain cross-file verification section heading")
		assert.Contains(t, content, "MUST read that file before asserting",
			"correctness.md must mandate reading files before asserting their contents")
		assert.Contains(t, content, "Do not reason about what a file",
			"correctness.md must prohibit reasoning about probable file contents")
	})

	t.Run("[test_id:TS-GH-1835-005] should prohibit presenting unverified content as fact", func(t *testing.T) {
		/*
			[NEGATIVE]
			Preconditions:
			    - correctness.md readable via FullsendRepoFile

			Steps:
			    1. Read correctness.md via FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
			    2. Search content for never-present prohibition
			    3. Search content for unable-to-verify fallback

			Expected:
			    - Content contains "Never present unverified file contents as fact"
			    - Content contains "unable to verify the contents" fallback language
		*/
		assert.Contains(t, content, "Never present unverified file contents as fact",
			"correctness.md must prohibit presenting unverified content as fact")
		assert.Contains(t, content, "unable to verify the contents",
			"correctness.md must contain fallback language for unverifiable files")
	})
}
