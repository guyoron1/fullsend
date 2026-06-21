package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Cross-File Verification — Code-Review SKILL.md Tests

STP Reference: outputs/stp/GH-1835/GH-1835_test_plan.md
Jira: GH-1835

Tests verify that the code-review SKILL.md contains cross-file
verification instructions that prevent the review agent from asserting
file contents it never read (GH-1835 incident).
*/

// normalizeWS collapses all runs of whitespace (including newlines) to
// a single space, making substring assertions immune to markdown line
// wrapping.
func normalizeWS(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func TestCrossFileVerificationSKILLMd(t *testing.T) {
	// Read SKILL.md once for all subtests — it's immutable embedded content.
	raw, err := FullsendRepoFile("skills/code-review/SKILL.md")
	require.NoError(t, err, "SKILL.md must be readable from scaffold embed")
	content := normalizeWS(string(raw))

	t.Run("[test_id:TS-GH-1835-001] should contain cross-file verification instruction in step 2", func(t *testing.T) {
		/*
		Preconditions:
		    - SKILL.md readable via FullsendRepoFile

		Steps:
		    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for cross-file verification label
		    3. Search content for MUST read mandate
		    4. Search content for never-claim instruction

		Expected:
		    - Content contains "Cross-file verification" heading or label in step 2
		    - Content contains "MUST read that file first" mandate
		    - Content contains "Never claim a file contains specific text without having read it"
		*/
		assert.Contains(t, content, "Cross-file verification",
			"SKILL.md must contain cross-file verification heading/label in step 2")
		assert.Contains(t, content, "MUST read that file first",
			"SKILL.md must mandate reading files before asserting their contents")
		assert.Contains(t, content, "Never claim a file contains specific text without having read it",
			"SKILL.md must prohibit claiming file contents without reading")
	})

	t.Run("[test_id:TS-GH-1835-002] should contain self-check gate in step 4", func(t *testing.T) {
		/*
		Preconditions:
		    - SKILL.md readable via FullsendRepoFile

		Steps:
		    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for self-check gate label
		    3. Search content for step 2 verification instruction
		    4. Search content for reframe instruction

		Expected:
		    - Content contains "Cross-file finding self-check" label in step 4
		    - Content contains instruction to "verify that you read that file during step 2"
		    - Content contains instruction to "reframe the finding" if file is unreadable
		*/
		assert.Contains(t, content, "Cross-file finding self-check",
			"SKILL.md must contain self-check gate label in step 4")
		assert.Contains(t, content, "verify that you read that file during step 2",
			"SKILL.md must instruct agent to verify step 2 file reads")
		assert.Contains(t, content, "reframe the finding",
			"SKILL.md must instruct agent to reframe findings for unreadable files")
	})

	t.Run("[test_id:TS-GH-1835-003] should contain unreadable file fallback language", func(t *testing.T) {
		/*
		[NEGATIVE]
		Preconditions:
		    - SKILL.md readable via FullsendRepoFile

		Steps:
		    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for "unable to verify" fallback language
		    3. Search content for "rather than assuming" prohibition

		Expected:
		    - Content contains "unable to verify the contents" fallback phrase
		    - Content contains "rather than assuming what they contain" prohibition
		*/
		assert.Contains(t, content, "unable to verify the contents",
			"SKILL.md must contain fallback language for unreadable files")
		assert.Contains(t, content, "rather than assuming what they contain",
			"SKILL.md must prohibit assuming contents of unreadable files")
	})

	t.Run("[test_id:TS-GH-1835-011] should require re-read if file not read in step 2", func(t *testing.T) {
		/*
		Preconditions:
		    - SKILL.md readable via FullsendRepoFile

		Steps:
		    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for re-read instruction in self-check gate

		Expected:
		    - Content contains "read it now before finalizing" instruction
		    - Instruction is positioned in step 4 self-check gate section
		*/
		assert.Contains(t, content, "read it now before finalizing",
			"SKILL.md must provide re-read remediation path in self-check gate")
	})

	t.Run("[test_id:TS-GH-1835-012] should reframe finding for unreadable files", func(t *testing.T) {
		/*
		[NEGATIVE]
		Preconditions:
		    - SKILL.md readable via FullsendRepoFile

		Steps:
		    1. Read SKILL.md via FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for reframe instruction for unreadable files
		    3. Search content for prohibition against asserting unverified content

		Expected:
		    - Content contains "reframe the finding" instruction
		    - Content contains "do not assert unverified contents as fact" prohibition
		*/
		assert.Contains(t, content, "reframe the finding",
			"SKILL.md must instruct agent to reframe findings for unreadable files")
		assert.Contains(t, content, "do not assert unverified contents as fact",
			"SKILL.md must prohibit asserting unverified contents as fact")
	})
}

// TestSKILLMdStepStructure verifies that the cross-file verification
// instructions appear in the correct structural positions within SKILL.md.
func TestSKILLMdStepStructure(t *testing.T) {
	raw, err := FullsendRepoFile("skills/code-review/SKILL.md")
	require.NoError(t, err)
	content := string(raw)

	t.Run("cross-file instruction appears before self-check gate", func(t *testing.T) {
		instructionIdx := strings.Index(content, "Cross-file verification")
		selfCheckIdx := strings.Index(content, "Cross-file finding self-check")
		if instructionIdx < 0 || selfCheckIdx < 0 {
			t.Skip("structural positions not found — content assertions cover this")
			return
		}
		assert.Less(t, instructionIdx, selfCheckIdx,
			"Cross-file verification instruction (step 2) must appear before self-check gate (step 4)")
	})
}
