package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Reconcile Flow Functional Tests — Update PR Lifecycle

STP Reference: outputs/stp/GH-2247/GH-2247_test_plan.md
Jira: GH-2247

End-to-end functional tests validating that the full reconcile-repos.sh flow
creates update PRs only for genuine content drift, and suppresses all API
activity when content matches.
*/

func TestReconcileFlow_UpdatePRLifecycle(t *testing.T) {
	t.Run("[test_id:TS-GH2247-011] update PR created for genuine template change", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote shim has user header + sentinel + stale content.
		remoteContent := "# Copyright 2026 Conforma\n# SPDX-License-Identifier: Apache-2.0\n" +
			sentinel + "\n" + staleTemplate + "\n"
		env.setRemoteContent(remoteContent)

		// Enhance mock to log detailed API calls for verification.
		ghCallsDetail := filepath.Join(env.tmpDir, "gh-calls-detail.log")
		enhanceMockGHForDetailedLogging(env, ghCallsDetail)

		output, err := env.run()
		_ = err

		// Verify stale detection triggered.
		assert.Contains(t, output, "shim is stale",
			"Script should detect stale content")

		// Verify full update flow executed.
		calls := env.ghCalls()
		callStr := strings.Join(calls, "\n")

		// Blob created.
		assert.True(t, env.blobCreated(),
			"Git blob should be created with fresh template content")

		// Tree created.
		assert.Contains(t, callStr, "git/trees",
			"Git tree should be created")

		// Commit created.
		assert.Contains(t, callStr, "git/commits",
			"Git commit should be created")

		// Branch ref created or updated.
		hasRefUpdate := strings.Contains(callStr, "git/refs")
		assert.True(t, hasRefUpdate,
			"Branch ref should be created or updated to point to new commit")

		// PR created (mock returns URL).
		assert.Contains(t, output, "pull/99",
			"Update PR should be created; output should contain PR URL")
	})

	t.Run("[test_id:TS-GH2247-012] no PR created when content matches", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content matches the template exactly.
		remoteContent := sentinel + "\n" + freshTemplate + "\n"
		env.setRemoteContent(remoteContent)

		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		// Verify no blob created.
		assert.False(t, env.blobCreated(),
			"No blob should be created when content matches")

		// Verify no git/blobs API call.
		for _, call := range env.ghCalls() {
			assert.False(t, strings.Contains(call, "git/blobs"),
				"No git/blobs API call should be made when content matches")
		}

		// Verify up-to-date message.
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"Script should log that the shim is up to date")
	})

	t.Run("[test_id:TS-GH2247-013] no blob created for false positive drift", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Remote content is identical to template but with encoding-only
		// differences (extra trailing newline). This produces different base64
		// but the decoded text comparison should recognize them as identical.
		remoteContent := sentinel + "\n" + freshTemplate + "\n\n"
		env.setRemoteContent(remoteContent)

		output, err := env.run()
		require.NoError(t, err, "reconcile-repos.sh should exit 0; output:\n%s", output)

		// Verify no blob created — the encoding-only difference should not
		// trigger any downstream API activity.
		assert.False(t, env.blobCreated(),
			"No blob should be created for encoding-only differences")

		// Double-check: no git/blobs endpoint hit.
		for _, call := range env.ghCalls() {
			assert.False(t, strings.Contains(call, "git/blobs"),
				"No git/blobs API call should be made for false positive drift; call: %s", call)
		}

		// The script should recognize content as up-to-date.
		assert.Contains(t, output, "already enrolled (shim up to date)",
			"Script should report content as up-to-date despite base64 differences")
	})
}

// enhanceMockGHForDetailedLogging adds more detailed logging to the mock gh
// script so functional tests can verify the complete API call sequence.
func enhanceMockGHForDetailedLogging(env *reconcileEnv, detailLog string) {
	env.t.Helper()

	mockPath := filepath.Join(env.mockBinDir, "gh")
	existing, err := os.ReadFile(mockPath)
	require.NoError(env.t, err)

	// Prepend detailed logging that includes method and endpoint.
	enhanced := strings.Replace(string(existing),
		fmt.Sprintf(`echo "$@" >> "%s"`, env.ghCallsLog),
		fmt.Sprintf(`echo "$@" >> "%s"
echo "$(date +%%s) $@" >> "%s"`, env.ghCallsLog, detailLog), 1)

	writeScript(env.t, mockPath, enhanced)
}
