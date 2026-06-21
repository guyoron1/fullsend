package scaffold

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Cross-File Verification — Code-Review SKILL.md Tests

STP Reference: outputs/stp/GH-1835/GH-1835_test_plan.md
Jira: GH-1835
*/

func TestCrossFileVerificationSKILLMd(t *testing.T) {
	/*
	Preconditions:
	    - PR #2443 changes merged into test branch
	    - code-review SKILL.md embedded via scaffold embed.FS
	*/

	t.Run("[test_id:TS-GH-1835-001] should contain cross-file verification instruction in step 2", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for cross-file verification label
		    3. Search content for MUST read mandate
		    4. Search content for never-claim instruction

		Expected:
		    - Content contains "Cross-file verification" heading or label in step 2
		    - Content contains "MUST read that file first" mandate
		    - Content contains "Never claim a file contains specific text without having read it"
		*/
		content, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "Cross-file verification"))
		assert.True(t, strings.Contains(s, "MUST read that file first"))
		assert.True(t, strings.Contains(s, "Never claim a file contains specific text without having read it"))
	})

	t.Run("[test_id:TS-GH-1835-002] should contain self-check gate in step 4", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for self-check gate label
		    3. Search content for step 2 verification instruction
		    4. Search content for reframe instruction

		Expected:
		    - Content contains "Cross-file finding self-check" label in step 4
		    - Content contains instruction to "verify that you read that file during step 2"
		    - Content contains instruction to "reframe the finding" if file is unreadable
		*/
		content, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "Cross-file finding self-check"))
		assert.True(t, strings.Contains(s, "verify that you read that file during step 2"))
		assert.True(t, strings.Contains(s, "reframe the finding"))
	})

	t.Run("[test_id:TS-GH-1835-003] should contain unreadable file fallback language", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for "unable to verify" fallback language
		    3. Search content for "rather than assuming" prohibition

		Expected:
		    - Content contains "unable to verify the contents" fallback phrase
		    - Content contains "rather than assuming what they contain" prohibition
		*/
		content, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "unable to verify the contents"))
		assert.True(t, strings.Contains(s, "rather than assuming what they contain"))
	})

	t.Run("[test_id:TS-GH-1835-011] should require re-read if file not read in step 2", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for re-read instruction in self-check gate

		Expected:
		    - Content contains "read it now before finalizing" instruction
		    - Instruction is positioned in step 4 self-check gate section
		*/
		content, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "read it now before finalizing"))
	})

	t.Run("[test_id:TS-GH-1835-012] should reframe finding for unreadable files", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		[NEGATIVE]
		Preconditions:
		    - SKILL.md readable via scaffold.FullsendRepoFile

		Steps:
		    1. Read SKILL.md via scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		    2. Search content for reframe instruction for unreadable files
		    3. Search content for prohibition against asserting unverified content

		Expected:
		    - Content contains "reframe the finding" instruction
		    - Content contains "do not assert unverified contents as fact" prohibition
		*/
		content, err := scaffold.FullsendRepoFile("skills/code-review/SKILL.md")
		require.NoError(t, err)

		s := string(content)
		assert.True(t, strings.Contains(s, "reframe the finding"))
		assert.True(t, strings.Contains(s, "do not assert unverified contents as fact"))
	})
}
