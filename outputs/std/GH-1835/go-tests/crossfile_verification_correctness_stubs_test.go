package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Cross-File Verification — Correctness Sub-Agent Tests

STP Reference: outputs/stp/GH-1835/GH-1835_test_plan.md
Jira: GH-1835
*/

func TestCrossFileVerificationCorrectnessSubAgent(t *testing.T) {
	/*
	Preconditions:
	    - PR #2443 changes merged into test branch
	    - correctness.md sub-agent embedded via scaffold embed.FS
	*/

	t.Run("[test_id:TS-GH-1835-004] should contain cross-file verification section", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - correctness.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read correctness.md via scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		    2. Search content for cross-file verification section heading
		    3. Search content for MUST read mandate
		    4. Search content for prohibition against probable reasoning

		Expected:
		    - Content contains "Cross-file verification" section heading
		    - Content contains "MUST read that file before asserting" mandate
		    - Content contains prohibition against reasoning about probable file contents
		*/
		content, err := scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "Cross-file verification"))
		assert.True(t, strings.Contains(s, "MUST read that file before asserting"))
		assert.True(t, strings.Contains(s, "Do not reason about what a file"))
	})

	t.Run("[test_id:TS-GH-1835-005] should prohibit presenting unverified content as fact", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - correctness.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read correctness.md via scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		    2. Search content for never-present prohibition
		    3. Search content for unable-to-verify fallback

		Expected:
		    - Content contains "Never present unverified file contents as fact"
		    - Content contains "unable to verify the contents" fallback language
		*/
		content, err := scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "Never present unverified file contents as fact"))
		assert.True(t, strings.Contains(s, "unable to verify the contents"))
	})
}
