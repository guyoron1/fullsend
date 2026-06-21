package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/scaffold"
)

/*
Cross-File Verification — Scaffold Embed & Deployment Tests

STP Reference: outputs/stp/GH-1835/GH-1835_test_plan.md
Jira: GH-1835
*/

func TestCrossFileVerificationScaffoldEmbed(t *testing.T) {
	/*
	Preconditions:
	    - PR #2443 changes merged into test branch
	    - Scaffold embed.FS includes updated skill files
	*/

	t.Run("[test_id:TS-GH-1835-006] should include SKILL.md in WalkFullsendRepo", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - scaffold.WalkFullsendRepo function available

		Steps:
		    1. Initialize path collector slice
		    2. Walk embedded filesystem via scaffold.WalkFullsendRepo
		    3. Collect all walked file paths
		    4. Search collected paths for skills/code-review/SKILL.md

		Expected:
		    - Walk completes without error
		    - Collected paths contain "skills/code-review/SKILL.md"
		*/
		var found []string
		err := scaffold.WalkFullsendRepo(func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			found = append(found, path)
			return nil
		})
		require.NoError(t, err)
		assert.Contains(t, found, "skills/code-review/SKILL.md")
	})

	t.Run("[test_id:TS-GH-1835-007] should include correctness.md in scaffold embed", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - scaffold.FullsendRepoFile function available

		Steps:
		    1. Read correctness.md from scaffold embed
		    2. Verify no error returned
		    3. Verify content is non-empty

		Expected:
		    - scaffold.FullsendRepoFile returns content without error
		    - Returned content has length > 0
		*/
		content, err := scaffold.FullsendRepoFile("skills/pr-review/sub-agents/correctness.md")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("[test_id:TS-GH-1835-008] should deploy updated files via scaffold install", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
		/*
		Preconditions:
		    - Writable temporary directory for install target
		    - scaffold.CollectInstallFiles function available

		Steps:
		    1. Create temporary directory via t.TempDir()
		    2. Run scaffold install to temporary directory
		    3. Read SKILL.md from target directory
		    4. Read correctness.md from target directory
		    5. Verify both files contain cross-file verification instructions

		Expected:
		    - Scaffold install writes SKILL.md to target directory
		    - Scaffold install writes correctness.md to target directory
		    - Written SKILL.md contains "Cross-file verification" text
		    - Written correctness.md contains "Cross-file verification" text
		*/
		tmpDir := t.TempDir()

		_ = tmpDir // scaffold install writes to tmpDir

		skillContent, err := os.ReadFile(filepath.Join(tmpDir, "skills/code-review/SKILL.md"))
		require.NoError(t, err)
		assert.True(t, strings.Contains(string(skillContent), "Cross-file verification"))

		correctnessContent, err := os.ReadFile(filepath.Join(tmpDir, "skills/pr-review/sub-agents/correctness.md"))
		require.NoError(t, err)
		assert.True(t, strings.Contains(string(correctnessContent), "Cross-file verification"))
	})
}
