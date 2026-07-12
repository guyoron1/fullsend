package layers

import (
	"context"
	"fmt"
	"time"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// WorkflowPollConfig configures the exponential-backoff polling used by
// AwaitWorkflowCompletion.
type WorkflowPollConfig struct {
	// InitialInterval is the delay before the first poll attempt.
	// Subsequent intervals double until reaching MaxInterval.
	InitialInterval time.Duration
	// MaxInterval caps the poll interval. If zero, no cap is applied.
	MaxInterval time.Duration
	// MaxAttempts is the maximum number of poll attempts before
	// returning a timeout error.
	MaxAttempts int
}

// AwaitWorkflowCompletion polls for a workflow run of workflowFile in
// org/repo that was created after dispatchTime, and waits for it to
// reach status "completed". Polling uses exponential backoff governed
// by cfg. Progress messages are sent to logFn (which may be nil).
func AwaitWorkflowCompletion(
	ctx context.Context,
	client forge.Client,
	org, repo, workflowFile string,
	dispatchTime time.Time,
	cfg WorkflowPollConfig,
	logFn func(string),
) (*forge.WorkflowRun, error) {
	interval := cfg.InitialInterval
	for attempt := range cfg.MaxAttempts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		runs, err := client.ListWorkflowRuns(ctx, org, repo, workflowFile)
		if err != nil {
			if logFn != nil {
				logFn(fmt.Sprintf("waiting for workflow run (attempt %d)...", attempt+1))
			}
			interval = nextBackoff(interval, cfg.MaxInterval)
			continue
		}

		for i := range runs {
			run := &runs[i]
			runTime, parseErr := time.Parse(time.RFC3339, run.CreatedAt)
			if parseErr != nil {
				continue
			}
			if runTime.Before(dispatchTime) {
				continue
			}

			if run.Status == "completed" {
				return run, nil
			}
			if logFn != nil {
				logFn(fmt.Sprintf("workflow run: %s (%s)", run.HTMLURL, run.Status))
			}
			break // found our run, keep waiting
		}
		interval = nextBackoff(interval, cfg.MaxInterval)
	}
	return nil, fmt.Errorf("timed out waiting for %s workflow", workflowFile)
}

// nextBackoff doubles the interval, capped at max.
// If max is zero, no cap is applied.
func nextBackoff(current, max time.Duration) time.Duration {
	next := current * 2
	if max > 0 && next > max {
		return max
	}
	return next
}
