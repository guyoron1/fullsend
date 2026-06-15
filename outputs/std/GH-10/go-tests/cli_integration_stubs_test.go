package cli

/*
runAgent Provider Integration Tests

STP Reference: outputs/stp/GH-10/GH-10_test_plan.md
Jira: GH-10

These stubs cover the CLI-level integration of the EnsureProvider
idempotency fix (#2294). They validate that runAgent in
internal/cli/run.go correctly handles provider lifecycle via
sandbox.EnsureProvider.
*/

import (
	"testing"
)

/*
Preconditions:
    - Test environment with fake gateway
    - A provider with the target name already exists (simulating prior run)

Steps:
    1. Execute runAgent which internally calls EnsureProvider

Expected:
    - runAgent returns nil error
*/
func TestRunAgent_ProviderAlreadyExists_Succeeds(t *testing.T) {
	// [test_id:TS-GH-10-009]
	t.Skip("Phase 1: Design only - awaiting implementation")
}

/*
[NEGATIVE]
Preconditions:
    - Test environment with openshell configured to fail on provider create
      with a non-AlreadyExists error

Steps:
    1. Execute runAgent which calls EnsureProvider

Expected:
    - runAgent returns a non-nil error
    - Error message is visible and descriptive
    - Error message does not contain secret values
*/
func TestRunAgent_ProviderCreateFails_AbortsWithClearError(t *testing.T) {
	// [test_id:TS-GH-10-010]
	t.Skip("Phase 1: Design only - awaiting implementation")
}

/*
Preconditions:
    - Clean test environment with no pre-existing providers

Steps:
    1. Execute first runAgent call (creates provider)
    2. Execute second runAgent call (provider already exists from step 1)

Expected:
    - First runAgent call completes without error
    - Second runAgent call completes without error (via delete-and-recreate)
    - No manual cleanup required between runs
*/
func TestRunAgent_SequentialCalls_ReuseProviders(t *testing.T) {
	// [test_id:TS-GH-10-011]
	t.Skip("Phase 1: Design only - awaiting implementation")
}
