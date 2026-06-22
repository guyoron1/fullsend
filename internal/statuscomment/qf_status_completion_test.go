package statuscomment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/config"
	"github.com/fullsend-ai/fullsend/internal/forge"
)

// QualityFlow tests for GH-71: Status Comment Failure Reporting
// STD Reference: outputs/std/GH-71/GH-71_test_description.yaml
// Scenarios: 17, 18, 19

func TestQF_PostCompletionExitCodeHandling(t *testing.T) {
	t.Run("[test_id:TS-GH-71-017] should post failure status comment on non-zero exit", func(t *testing.T) {
		// Scenario 17: PostCompletion with failure status posts failure indicator.
		// When the agent exits non-zero, PostCompletion is called with status="failure".
		// The resulting comment must contain a failure emoji and the word "Failure".
		fc := forge.NewFakeClient()
		cfg := config.StatusNotificationConfig{
			Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
		}
		n := newTestNotifier(fc, cfg)

		err := n.PostStart(context.Background(), "Coding issue #71")
		require.NoError(t, err)
		require.Equal(t, 1, n.startCommentID)

		completionTime := fixedTime().Add(8 * time.Minute)
		n.now = func() time.Time { return completionTime }

		// Simulate non-zero exit: PostCompletion is called with status "failure"
		err = n.PostCompletion(context.Background(), "Coding issue #71", "failure")
		require.NoError(t, err)

		// ASSERT-01: Status comment indicates failure
		require.Len(t, fc.UpdatedComments, 1, "should update the start comment with completion")
		body := fc.UpdatedComments[0].Body
		assert.Contains(t, body, "❌ Failure", "comment must contain failure emoji and label")
		assert.Contains(t, body, "Finished Coding issue #71", "comment must reference the description")
		assert.Contains(t, body, "[View workflow run →](https://ci/run/42)", "comment must include workflow run link")
		assert.Contains(t, body, terminalTag, "comment must include terminal tag")
	})

	t.Run("[test_id:TS-GH-71-018] should post success status comment on zero exit", func(t *testing.T) {
		// Scenario 18: PostCompletion with success status posts success indicator.
		// When the agent exits with code 0, PostCompletion is called with status="success".
		fc := forge.NewFakeClient()
		cfg := config.StatusNotificationConfig{
			Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
		}
		n := newTestNotifier(fc, cfg)

		err := n.PostStart(context.Background(), "Coding issue #71")
		require.NoError(t, err)

		completionTime := fixedTime().Add(5 * time.Minute)
		n.now = func() time.Time { return completionTime }

		// Simulate successful exit: PostCompletion called with status "success"
		err = n.PostCompletion(context.Background(), "Coding issue #71", "success")
		require.NoError(t, err)

		// ASSERT-01: Status comment indicates success
		require.Len(t, fc.UpdatedComments, 1)
		body := fc.UpdatedComments[0].Body
		assert.Contains(t, body, "✅ Success", "comment must contain success emoji and label")
		assert.Contains(t, body, "Finished Coding issue #71")
		assert.Contains(t, body, terminalTag)
	})

	t.Run("[test_id:TS-GH-71-019] should post cancelled status on context cancellation", func(t *testing.T) {
		// Scenario 19: PostCompletion with cancelled status posts cancellation indicator.
		// When the agent run is cancelled (e.g., user-initiated, timeout), PostCompletion
		// is called with status="cancelled". The cancelled state must be distinguishable
		// from both success and failure.
		fc := forge.NewFakeClient()
		cfg := config.StatusNotificationConfig{
			Comment: config.CommentNotificationConfig{Start: "enabled", Completion: "enabled"},
		}
		n := newTestNotifier(fc, cfg)

		err := n.PostStart(context.Background(), "Coding issue #71")
		require.NoError(t, err)

		completionTime := fixedTime().Add(3 * time.Minute)
		n.now = func() time.Time { return completionTime }

		// Simulate cancellation: PostCompletion called with status "cancelled"
		err = n.PostCompletion(context.Background(), "Coding issue #71", "cancelled")
		require.NoError(t, err)

		// ASSERT-01: Status comment indicates cancellation
		require.Len(t, fc.UpdatedComments, 1)
		body := fc.UpdatedComments[0].Body
		assert.Contains(t, body, "⚠️ Cancelled", "comment must contain cancellation emoji and label")
		assert.Contains(t, body, "Finished Coding issue #71")
		assert.Contains(t, body, terminalTag)

		// Verify cancelled is distinct from success and failure
		assert.NotContains(t, body, "✅", "cancelled must not show success emoji")
		assert.NotContains(t, body, "❌", "cancelled must not show failure emoji")
	})
}
