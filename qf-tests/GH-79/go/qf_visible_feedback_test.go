package dispatch_auth

import (
	"testing"
)

/*
Visible Feedback Tests (BLOCKED)

STP Reference: outputs/stp/GH-79/GH-79_test_plan.md
STD Reference: outputs/std/GH-79/GH-79_test_description.yaml
Jira: GH-79

These tests are BLOCKED because visible feedback (reaction or comment on
unauthorized slash command attempts) is not implemented in this PR. ADR 0051
requires it for a future implementation.
*/

func TestVisibleFeedback(t *testing.T) {

	t.Run("unauthorized slash command attempt produces visible feedback", func(t *testing.T) {
		// [test_id:TS-GH-79-036] P1 BLOCKED
		// Blocked reason: Visible feedback not implemented in this PR —
		// ADR 0051 requires it for future implementation.
		t.Skip("BLOCKED: Visible feedback not implemented in this PR — ADR 0051 requires it for future implementation")
	})

	t.Run("unauthorized PR-triggered dispatch produces visible feedback", func(t *testing.T) {
		// [test_id:TS-GH-79-037] P1 BLOCKED
		// Blocked reason: Visible feedback not implemented in this PR.
		t.Skip("BLOCKED: Visible feedback not implemented in this PR")
	})
}
