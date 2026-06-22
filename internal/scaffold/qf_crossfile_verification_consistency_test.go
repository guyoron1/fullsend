package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Cross-File Verification — Cross-File Consistency Tests

STP Reference: outputs/stp/GH-70/GH-70_test_plan.md
STD Reference: outputs/std/GH-70/GH-70_test_description.yaml
Jira: GH-1835

Tests verify consistency of cross-file verification language across
both the code-review SKILL.md and the correctness sub-agent definition.
Ensures uniform behavior regardless of which component encounters an
unreadable file.
*/

func TestQFCrossFileVerificationConsistency(t *testing.T) {
	// Read both files once for all consistency subtests.
	// qfNormalizeWS (defined in qf_crossfile_verification_skill_test.go) collapses
	// whitespace so assertions are immune to markdown line wrapping.
	skillRaw, err := FullsendRepoFile("skills/code-review/SKILL.md")
	require.NoError(t, err, "SKILL.md must be readable from scaffold embed")
	skillContent := qfNormalizeWS(string(skillRaw))

	correctnessRaw, err := FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
	require.NoError(t, err, "correctness.md must be readable from scaffold embed")
	correctnessContent := qfNormalizeWS(string(correctnessRaw))

	t.Run("[test_id:TS-GH-1835-009] should contain graceful degradation language in both skill files", func(t *testing.T) {
		/*
			Preconditions:
			    - SKILL.md readable via FullsendRepoFile
			    - correctness.md readable via FullsendRepoFile

			Steps:
			    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
			    2. Read correctness.md via FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
			    3. Search SKILL.md for graceful degradation language
			    4. Search correctness.md for graceful degradation language

			Expected:
			    - SKILL.md contains "unable to verify the contents" phrasing
			    - correctness.md contains "unable to verify the contents" phrasing
			    - Both files use consistent fallback language
		*/
		assert.Contains(t, skillContent, "unable to verify the contents",
			"SKILL.md must contain graceful degradation language")
		assert.Contains(t, correctnessContent, "unable to verify the contents",
			"correctness.md must contain graceful degradation language")
	})

	t.Run("[test_id:TS-GH-1835-010] should not contain assumption phrasing in either file", func(t *testing.T) {
		/*
			[NEGATIVE]
			Preconditions:
			    - SKILL.md readable via FullsendRepoFile
			    - correctness.md readable via FullsendRepoFile

			Steps:
			    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
			    2. Read correctness.md via FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
			    3. Search SKILL.md for prohibited "assume the contents" phrasing
			    4. Search correctness.md for prohibited "assume the contents" phrasing

			Expected:
			    - SKILL.md does NOT contain "assume the contents" phrasing
			    - correctness.md does NOT contain "assume the contents" phrasing
		*/
		assert.NotContains(t, skillContent, "assume the contents",
			"SKILL.md must not contain assumption phrasing")
		assert.NotContains(t, correctnessContent, "assume the contents",
			"correctness.md must not contain assumption phrasing")
	})
}
