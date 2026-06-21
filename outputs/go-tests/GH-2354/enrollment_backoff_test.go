package layers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
Enrollment Exponential Backoff Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.2 Exponential Backoff
*/

func TestEnrollmentBackoff(t *testing.T) {
	t.Run("[test_id:TS-GH2354-005] Polling interval doubles from initial to max", func(t *testing.T) {
		// Table-driven test covering the full backoff progression:
		// 2s → 4s → 8s → 15s (capped).
		tests := []struct {
			name     string
			current  time.Duration
			expected time.Duration
		}{
			{"doubles small interval", 2 * time.Second, 4 * time.Second},
			{"doubles again", 4 * time.Second, 8 * time.Second},
			{"caps at max", 8 * time.Second, enrollmentPollMax},
			{"stays at max", enrollmentPollMax, enrollmentPollMax},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := nextInterval(tt.current)
				assert.Equal(t, tt.expected, got)
			})
		}
	})

	t.Run("[test_id:TS-GH2354-006] nextInterval caps at enrollmentPollMax", func(t *testing.T) {
		// Verify the cap works at and above enrollmentPollMax.
		got := nextInterval(enrollmentPollMax)
		assert.Equal(t, enrollmentPollMax, got, "at cap should return cap")

		gotOver := nextInterval(enrollmentPollMax + 5*time.Second)
		assert.Equal(t, enrollmentPollMax, gotOver, "above cap should return cap")
	})

	t.Run("[test_id:TS-GH2354-007] nextInterval doubles sub-max values", func(t *testing.T) {
		// Verify each sub-max value doubles correctly.
		assert.Equal(t, 4*time.Second, nextInterval(2*time.Second))
		assert.Equal(t, 8*time.Second, nextInterval(4*time.Second))
		assert.Equal(t, enrollmentPollMax, nextInterval(8*time.Second))
	})
}
