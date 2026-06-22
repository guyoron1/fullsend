package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/statuscomment"
)

// QualityFlow tests for GH-71: Reconcile-Status Command
// STD Reference: outputs/std/GH-71/GH-71_test_description.yaml
// Scenarios: 20, 21, 22
//
// These tests validate the reconcile-status command's ability to finalize
// orphaned status comments with the correct reason (terminated, cancelled)
// and its idempotency when comments are already finalized.
//
// Tests call statuscomment.ReconcileOrphaned directly with forge.FakeClient,
// which is the same code path executed by the reconcile-status CLI command.

func TestQF_ReconcileStatusOrphanedComments(t *testing.T) {
	t.Run("[test_id:TS-GH-71-020] should finalize orphaned comment as terminated", func(t *testing.T) {
		// Scenario 20: When an in-progress status comment is found with no
		// corresponding active run, reconcile-status finalizes it with
		// "terminated" status.
		fc := forge.NewFakeClient()
		fc.IssueComments = map[string][]forge.IssueComment{
			"org/repo/7": {
				{
					ID:     42,
					Body:   "<!-- fullsend:agent-status:run-99 -->\n🤖 Code · Started 6:43 AM UTC\nCommit: `abc1234`",
					Author: "fullsend-bot[bot]",
				},
			},
		}

		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 7,
			"run-99", "https://ci/run/99", "abc1234def",
			statuscomment.ReasonTerminated,
		)
		require.NoError(t, err)

		// ASSERT-01: Orphaned comment finalized as terminated
		require.Len(t, fc.UpdatedComments, 1)
		body := fc.UpdatedComments[0].Body
		assert.Equal(t, 42, fc.UpdatedComments[0].CommentID)
		assert.Contains(t, body, "❌ Terminated", "must show terminated status")
		assert.Contains(t, body, "Code", "must preserve original description")
		assert.Contains(t, body, "Started 6:43 AM UTC", "must preserve start time")
		assert.Contains(t, body, "<!-- fullsend:status:terminal -->", "must add terminal tag")
	})

	t.Run("[test_id:TS-GH-71-021] should finalize orphaned comment as cancelled", func(t *testing.T) {
		// Scenario 21: When reconcile-status is called with reason=cancelled,
		// the orphaned comment is finalized with a "cancelled" status instead
		// of "terminated".
		fc := forge.NewFakeClient()
		fc.IssueComments = map[string][]forge.IssueComment{
			"org/repo/7": {
				{
					ID:     42,
					Body:   "<!-- fullsend:agent-status:run-99 -->\n🤖 Reviewing this PR · Started 2:34 PM UTC\nCommit: `abc1234`",
					Author: "fullsend-bot[bot]",
				},
			},
		}

		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 7,
			"run-99", "https://ci/run/99", "abc1234def",
			statuscomment.ReasonCancelled,
		)
		require.NoError(t, err)

		// ASSERT-01: Comment finalized as cancelled
		require.Len(t, fc.UpdatedComments, 1)
		body := fc.UpdatedComments[0].Body
		assert.Contains(t, body, "⚠️ Cancelled", "must show cancelled status")
		assert.Contains(t, body, "Reviewing this PR", "must preserve description")
		assert.Contains(t, body, "Started 2:34 PM UTC", "must preserve start time")
		assert.Contains(t, body, "<!-- fullsend:status:terminal -->", "must add terminal tag")
	})

	t.Run("[test_id:TS-GH-71-022] should be idempotent when comment already finalized", func(t *testing.T) {
		// Scenario 22: When the status comment has already been finalized
		// (contains terminal tag), running reconcile-status again must NOT
		// modify it. This ensures idempotency.
		fc := forge.NewFakeClient()
		fc.IssueComments = map[string][]forge.IssueComment{
			"org/repo/7": {
				{
					ID:     42,
					Body:   "<!-- fullsend:agent-status:run-99 -->\n<!-- fullsend:status:terminal -->\n🤖 Finished Code · ✅ Success · Started 6:43 AM UTC · Completed 6:50 AM UTC",
					Author: "fullsend-bot[bot]",
				},
			},
		}

		err := statuscomment.ReconcileOrphaned(
			context.Background(), fc, "org", "repo", 7,
			"run-99", "https://ci/run/99", "abc1234def",
			statuscomment.ReasonTerminated,
		)
		require.NoError(t, err)

		// ASSERT-01: Already-finalized comment is not modified
		assert.Empty(t, fc.UpdatedComments, "must not update already-finalized comment")
	})
}
