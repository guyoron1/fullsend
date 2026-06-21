package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Cross-File Verification — Cross-File Consistency Tests

STP Reference: outputs/stp/GH-1835/GH-1835_test_plan.md
Jira: GH-1835
*/

func TestCrossFileVerificationConsistency(t *testing.T) {
	/*
	Preconditions:
	    - PR #2443 changes merged into test branch
	    - Both SKILL.md and correctness.md embedded via scaffold embed.FS
	*/

	t.Run("[test_id:TS-GH-1835-009] should contain graceful degradation language in both skill files", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile
		    - correctness.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Read correctness.md via scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		    3. Search SKILL.md for graceful degradation language
		    4. Search correctness.md for graceful degradation language

		Expected:
		    - SKILL.md contains "unable to verify the contents" phrasing
		    - correctness.md contains "unable to verify the contents" phrasing
		    - Both files use consistent fallback language
		*/
		skillRaw, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)
		skillContent := string(skillRaw)

		correctnessRaw, err := scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err)
		correctnessContent := string(correctnessRaw)

		assert.True(t, strings.Contains(skillContent, "unable to verify the contents"))
		assert.True(t, strings.Contains(correctnessContent, "unable to verify the contents"))
	})

	t.Run("[test_id:TS-GH-1835-010] should not contain assumption phrasing in either file", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile
		    - correctness.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Read correctness.md via scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		    3. Search SKILL.md for prohibited "assume the contents" phrasing
		    4. Search correctness.md for prohibited "assume the contents" phrasing

		Expected:
		    - SKILL.md does NOT contain "assume the contents" phrasing
		    - correctness.md does NOT contain "assume the contents" phrasing
		*/
		skillRaw, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)
		skillContent := string(skillRaw)

		correctnessRaw, err := scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err)
		correctnessContent := string(correctnessRaw)

		assert.False(t, strings.Contains(skillContent, "assume the contents"))
		assert.False(t, strings.Contains(correctnessContent, "assume the contents"))
	})
}
