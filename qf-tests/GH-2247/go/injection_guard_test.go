//go:build e2e

package scaffold_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Content Injection Guard Tests

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

Requirement: Non-comment YAML content above the sentinel is rejected to prevent
workflow injection. Only comment lines (starting with #) and empty lines are
allowed above the sentinel boundary.
*/

// ghMockForInjectionTest returns a standard gh mock case body for injection
// guard tests with the given remote content base64.
func ghMockForInjectionTest(remoteB64, tmpDir string) string {
	return `
case "$endpoint" in
  repos/test-org/test-repo/actions/variables/*)
    json='{"status":"404","message":"Not Found"}'
    rc=1
    ;;
  repos/test-org/test-repo/contents/.github/workflows/fullsend.yaml)
    json='{"content":"` + remoteB64 + `","sha":"file-sha"}'
    ;;
  repos/test-org/test-repo)
    json='{"default_branch":"main","private":false}'
    ;;
  repos/test-org/test-repo/git/ref/heads/*)
    json='{"object":{"sha":"base-sha"}}'
    ;;
  repos/test-org/test-repo/git/commits/base-sha)
    json='{"tree":{"sha":"base-tree-sha"}}'
    ;;
  repos/test-org/test-repo/git/blobs)
    json='{"sha":"blob-sha"}'
    ;;
  repos/test-org/test-repo/git/trees)
    json='{"sha":"tree-sha"}'
    ;;
  repos/test-org/test-repo/git/commits)
    json='{"sha":"desired-commit-sha"}'
    ;;
  repos/test-org/test-repo/git/refs)
    rc=1
    ;;
  repos/test-org/test-repo/git/refs/heads/*)
    rc=0
    ;;
  *)
    rc=0
    ;;
esac
`
}

func TestInjectionGuard(t *testing.T) {

	/*
	TS-GH-2247-013 — Verify non-comment YAML content above sentinel is rejected.
	An attacker could inject malicious workflow steps via content above sentinel.
	*/
	t.Run("[test_id:TS-GH-2247-013]_non_comment_yaml_rejected", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote has non-comment YAML ("name: injected-workflow") above sentinel.
		remoteContent := "name: injected-workflow\n" + sentinelLine + "\nstale shim template\n"
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, ghMockForInjectionTest(remoteB64, env.tmpDir))

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for shim update")

		// Injected content must NOT appear in output blob.
		assert.NotContains(t, blob, "injected-workflow",
			"non-comment YAML above sentinel must be rejected from output")

		// Sentinel and fresh template should still be present.
		assert.Contains(t, blob, sentinelLine,
			"sentinel must be present in output")
		assert.Contains(t, blob, freshTemplate,
			"fresh template must be present in output")
	})

	/*
	TS-GH-2247-014 — Verify warning is emitted to stderr when injection guard
	rejects non-comment content.
	*/
	t.Run("[test_id:TS-GH-2247-014]_warning_emitted_on_rejection", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote with non-comment content above sentinel.
		remoteContent := "name: injected-workflow\n" + sentinelLine + "\nstale shim template\n"
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, ghMockForInjectionTest(remoteB64, env.tmpDir))

		output, _ := env.runReconcile(t)

		// Should emit a warning about rejected header content.
		assert.Contains(t, output, "non-comment content above sentinel was rejected",
			"warning should be emitted when injection guard rejects content")
	})

	/*
	TS-GH-2247-015 — Verify dangerous workflow keys (jobs:, steps:, run:) placed
	above sentinel do not appear in the output blob's header section.
	Defense-in-depth check complementing scenario 13.
	*/
	t.Run("[test_id:TS-GH-2247-015]_injected_workflow_keys_absent_from_output", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote with multiple dangerous workflow keys above sentinel.
		injectedHeader := "jobs:\n  inject:\n    steps:\n    - run: echo malicious\n"
		remoteContent := injectedHeader + sentinelLine + "\nstale shim template\n"
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, ghMockForInjectionTest(remoteB64, env.tmpDir))

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for shim update")

		// Extract header portion (above sentinel).
		sentinelIdx := strings.Index(blob, sentinelLine)
		require.True(t, sentinelIdx >= 0, "blob should contain sentinel")
		header := blob[:sentinelIdx]

		dangerousKeys := []string{"jobs:", "steps:", "run:"}
		for _, key := range dangerousKeys {
			assert.NotContains(t, header, key,
				"dangerous workflow key %q must not appear in output header", key)
		}
	})

	/*
	TS-GH-2247-016 — Verify YAML document separator (---) above sentinel is
	treated as non-comment content and rejected.
	*/
	t.Run("[test_id:TS-GH-2247-016]_yaml_document_separator_treated_as_non_comment", func(t *testing.T) {
		env := newTestEnv(t)

		// Remote with only "---" above sentinel.
		remoteContent := "---\n" + sentinelLine + "\nstale shim template\n"
		remoteB64 := encodeB64(remoteContent)

		env.installGhMock(t, ghMockForInjectionTest(remoteB64, env.tmpDir))

		_, _ = env.runReconcile(t)

		blob, ok := env.blobContent(t, "test-repo")
		require.True(t, ok, "blob should be created for shim update")

		// Extract header portion.
		sentinelIdx := strings.Index(blob, sentinelLine)
		require.True(t, sentinelIdx >= 0, "blob should contain sentinel")
		header := blob[:sentinelIdx]

		assert.NotContains(t, header, "---",
			"YAML document separator must be rejected as non-comment content")
	})
}
