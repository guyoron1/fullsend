package github_test

/*
isTransientStatus Tests

STP Reference: outputs/stp/GH-24/GH-24_test_plan.md
Jira: GH-24

Tests that isTransientStatus returns true only for 404 and 409, and returns
false for all 5xx status codes (which are now handled by do() retries).
*/

import (
	"testing"
)

/*
Markers:
    - tier1

Preconditions:
    - Go toolchain 1.23+
    - Module dependencies resolved
*/

// TestIsTransientStatus_Only404And409 validates narrowed transient status scope
func TestIsTransientStatus_Only404And409(t *testing.T) {

	/*
	Preconditions: None

	Steps:
	    1. Call isTransientStatus(404)

	Expected:
	    - Returns true
	*/
	t.Run("[test_id:TS-GH-24-021] should return true for 404", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
	})

	/*
	Preconditions: None

	Steps:
	    1. Call isTransientStatus(409)

	Expected:
	    - Returns true
	*/
	t.Run("[test_id:TS-GH-24-022] should return true for 409", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
	})

	/*
	[NEGATIVE]
	Preconditions: None

	Steps:
	    1. Call isTransientStatus for each 5xx code (500, 502, 503, 504)

	Expected:
	    - Returns false for 500
	    - Returns false for 502
	    - Returns false for 503
	    - Returns false for 504
	*/
	t.Run("[test_id:TS-GH-24-023] should return false for 500 502 503 504", func(t *testing.T) {
		t.Skip("Phase 1: Design only - awaiting implementation")
	})
}
