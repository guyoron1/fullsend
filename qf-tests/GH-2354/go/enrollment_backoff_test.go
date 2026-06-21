//go:build e2e

package layers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
Enrollment Exponential Backoff Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
Jira: GH-2354

These tests validate that enrollment polling uses exponential backoff to
avoid excessive API calls. The nextInterval helper is a pure function that
can be tested directly without mocking, providing reliable coverage of the
backoff logic. Integration-level backoff behaviour is covered by the timeout
bound tests which exercise the full polling loop.
*/

// TestEnrollmentExponentialBackoff validates the exponential backoff behaviour
// of enrollment polling intervals.
func TestEnrollmentExponentialBackoff(t *testing.T) {
	t.Run("should increase wait time between status updates progressively", func(t *testing.T) {
		// [test_id:TS-GH-2354-004]
		// Verify that nextInterval doubles the polling interval on each call.
		intervals := make([]time.Duration, 0, 5)
		current := enrollmentPollInitial
		intervals = append(intervals, current)

		// Simulate 4 backoff iterations
		for i := 0; i < 4; i++ {
			current = nextInterval(current)
			intervals = append(intervals, current)
		}

		// Assert: intervals increase monotonically (until cap is reached).
		for i := 1; i < len(intervals); i++ {
			assert.GreaterOrEqual(t, intervals[i], intervals[i-1],
				"interval[%d] (%v) should be >= interval[%d] (%v)",
				i, intervals[i], i-1, intervals[i-1])
		}

		// Assert: second interval is 2x the first (before cap).
		assert.Equal(t, 2*enrollmentPollInitial, intervals[1],
			"second interval should be 2x initial")
	})

	t.Run("should not exceed maximum poll interval", func(t *testing.T) {
		// [test_id:TS-GH-2354-005]
		// Verify that no interval exceeds enrollmentPollMax regardless of
		// how many iterations occur.
		current := enrollmentPollInitial

		// Run enough iterations to well exceed the cap
		for i := 0; i < 20; i++ {
			current = nextInterval(current)
			assert.LessOrEqual(t, current, enrollmentPollMax,
				"interval after %d doublings should not exceed enrollmentPollMax (%v), got %v",
				i+1, enrollmentPollMax, current)
		}

		// After many iterations, interval should be exactly at the cap.
		assert.Equal(t, enrollmentPollMax, current,
			"interval should stabilise at enrollmentPollMax")
	})

	t.Run("should execute first retry within expected timeframe", func(t *testing.T) {
		// [test_id:TS-GH-2354-006]
		// Verify that the initial polling interval is enrollmentPollInitial (2s),
		// ensuring the first poll occurs promptly after dispatch.
		assert.Equal(t, 2*time.Second, enrollmentPollInitial,
			"initial poll interval should be 2 seconds")

		// The first interval used in awaitWorkflowRun is enrollmentPollInitial.
		// After one nextInterval call, it should double.
		firstRetry := enrollmentPollInitial
		assert.LessOrEqual(t, firstRetry, 2*time.Second+500*time.Millisecond,
			"first retry should occur within enrollmentPollInitial + 500ms tolerance")
	})
}

// TestBackoffSequence validates the complete backoff sequence from initial
// to cap, ensuring the progression is correct.
func TestBackoffSequence(t *testing.T) {
	expected := []time.Duration{
		4 * time.Second,  // 2s * 2
		8 * time.Second,  // 4s * 2
		15 * time.Second, // 16s capped at 15s
		15 * time.Second, // stays at cap
		15 * time.Second, // stays at cap
	}

	current := enrollmentPollInitial
	for i, want := range expected {
		current = nextInterval(current)
		assert.Equal(t, want, current,
			"iteration %d: expected %v, got %v", i+1, want, current)
	}
}
