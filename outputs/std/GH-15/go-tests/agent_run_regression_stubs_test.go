package sandbox

/*
Agent Run Regression Tests — EnsureProvider Idempotency

STP Reference: outputs/stp/GH-15/GH-15_test_plan.md
Jira: GH-15
*/

import (
	"testing"
)

/*
Markers:
    - regression

Preconditions:
    - Go 1.23+ toolchain available
    - Fake openshell binary in $TMPDIR to simulate CLI behavior
    - Agent configuration with multiple provider definitions
*/

/*
Preconditions:
    - Fake openshell script that returns AlreadyExists for all provider
      creates, succeeds on deletes and retry creates
    - Agent configured with multiple provider definitions
    - Temporary directory prepended to PATH

Steps:
    1. Call runAgent with the test configuration

Expected:
    - runAgent completes without error
    - All providers are successfully recreated
    - Agent proceeds to sandbox operations after provider setup
*/
func TestRunAgent_PreExistingProviders_Succeeds(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation [test_id:TS-GH-15-013]")
}

/*
[NEGATIVE]
Preconditions:
    - Fake openshell script that fails with a non-AlreadyExists error
      on provider create
    - Agent configured with provider definitions
    - Temporary directory prepended to PATH

Steps:
    1. Call runAgent with the test configuration

Expected:
    - runAgent returns an error
    - Error message includes the provider name and failure reason
    - Subsequent providers are not attempted after failure
*/
func TestRunAgent_NonIdempotentError_FailsFast(t *testing.T) {
	t.Skip("Phase 1: Design only - awaiting implementation [test_id:TS-GH-15-014]")
}
