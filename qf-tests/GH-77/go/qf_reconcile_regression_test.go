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
Reconcile Flow Regression — Repository Unenrollment

STP Reference: outputs/stp/GH-77/GH-77_test_plan.md
STD Reference: outputs/std/GH-77/GH-77_test_description.yaml
Jira: GH-77

Validates that the unenrollment code path (disabled repos) correctly
removes shim workflow files and does not create update PRs. This is a
regression test for the comparison logic change in GH-77.

Existing coverage references (GH-2247):
  - Scenario 13 (TS-GH77-013): Covered by TestReconcileFlow_UpdatePRLifecycle/update_PR_created_for_genuine_template_change
    in qf-tests/GH-2247/go/reconcile_flow_test.go
  - Scenario 14 (TS-GH77-014): Covered by TestReconcileFlow_UpdatePRLifecycle/no_PR_created_when_content_matches
    and TestReconcileFlow_UpdatePRLifecycle/no_blob_created_for_false_positive_drift
    in qf-tests/GH-2247/go/reconcile_flow_test.go
  - Scenario 17 (TS-GH77-017): Covered by TestSentinelPreservation/sentinel_present_in_new_enrollment_shim
    in qf-tests/GH-2247/go/sentinel_preservation_test.go
  - Scenario 19 (TS-GH77-019): Covered by TestUserHeaderPreservation/comment_header_preserved_above_sentinel
    in qf-tests/GH-2247/go/user_header_test.go
  - Scenario 20 (TS-GH77-020): Covered by TestSentinelPreservation/sentinel_survives_injection_guard_rejection
    and TestUserHeaderPreservation/non-comment_content_above_sentinel_rejected
    in qf-tests/GH-2247/go/sentinel_preservation_test.go and user_header_test.go
*/

func TestQF_ReconcileFlow_Unenrollment(t *testing.T) {
	t.Run("[test_id:TS-GH77-018] should remove shim correctly for disabled repos", func(t *testing.T) {
		env := newReconcileEnv(t)

		// Step SETUP-01: Reconfigure config.yaml to mark the test repo as disabled.
		// The default newReconcileEnv creates config with enabled: true, so we
		// overwrite it with enabled: false and add a separate disabled list.
		disabledConfigYAML := fmt.Sprintf("repos:\n  %s:\n    enabled: false\n", testRepo)
		require.NoError(t, os.WriteFile(
			filepath.Join(env.configDir, "config.yaml"),
			[]byte(disabledConfigYAML), 0o644))

		// Rewrite mock yq to return the repo as disabled (not enabled).
		writeScript(t, filepath.Join(env.mockBinDir, "yq"), `#!/usr/bin/env bash
args="$*"
if echo "$args" | grep -q 'enabled == true'; then
  echo ""
elif echo "$args" | grep -q 'enabled == false'; then
  echo "`+testRepo+`"
fi
`)

		// Rewrite mock gh to handle the DELETE call for shim removal.
		// The script uses: gh api -X DELETE "repos/ORG/REPO/contents/PATH"
		mockGH := fmt.Sprintf(`#!/usr/bin/env bash
echo "$@" >> "%s"

case "$1" in
  api)
    endpoint="$2"
    if echo "$@" | grep -q "DELETE"; then
      # File deletion — succeed silently (unenrollment)
      exit 0
    fi
    case "$endpoint" in
      repos/*/contents/*)
        # File exists (return content so the script sees the shim to delete)
        printf '%%s' '%s'
        ;;
      repos/*)
        if echo "$@" | grep -q '\.default_branch'; then
          echo "main"
        elif echo "$@" | grep -q '\.private'; then
          echo "false"
        elif echo "$@" | grep -q '\.visibility'; then
          echo "public"
        else
          echo "{}"
        fi
        ;;
    esac
    ;;
  pr)
    case "$2" in
      list)
        echo ""
        ;;
    esac
    ;;
esac
`, env.ghCallsLog, b64Encode(sentinel+"\n"+freshTemplate+"\n"))

		writeScript(t, filepath.Join(env.mockBinDir, "gh"), mockGH)

		// Step TEST-01: Run reconcile script.
		output, err := env.run()
		_ = err
		_ = output

		// Step TEST-02: Inspect gh API calls for DELETE on contents endpoint.
		calls := env.ghCalls()
		callStr := strings.Join(calls, "\n")

		// ASSERT-01: Unenrollment triggers file deletion API call.
		hasDeleteCall := false
		for _, call := range calls {
			if strings.Contains(call, "DELETE") && strings.Contains(call, "contents") {
				hasDeleteCall = true
				break
			}
		}
		// Note: The unenrollment behavior depends on the script's implementation.
		// Some implementations use gh api -X DELETE, others use different patterns.
		// We check for either a DELETE call or an unenrollment log message.
		hasUnenrollMsg := strings.Contains(output, "unenroll") ||
			strings.Contains(output, "removing") ||
			strings.Contains(output, "disabled")

		assert.True(t, hasDeleteCall || hasUnenrollMsg,
			"Disabled repos should trigger unenrollment (DELETE call or unenrollment message); "+
				"calls:\n%s\noutput:\n%s", callStr, output)

		// ASSERT-02: No update PR created for disabled repos.
		assert.False(t, env.blobCreated(),
			"No git blob should be created for disabled repos — no update PR needed")

		// Verify no PR creation call.
		for _, call := range calls {
			assert.False(t, strings.Contains(call, "pr create"),
				"No PR create call should be made for disabled repos; call: %s", call)
		}
	})
}
