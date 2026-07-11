package layers

import (
	"context"
	"fmt"
	"time"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// PollConfig configures workflow polling behavior.
type PollConfig struct {
	// InitialInterval is the delay before the first poll and the starting
	// interval for exponential backoff.
	InitialInterval time.Duration

	// MaxInterval caps how long the backoff can grow between polls.
	MaxInterval time.Duration

	// Timeout is the maximum wall-clock time to wait for the workflow to
	// complete before returning an error.
	Timeout time.Duration
}

// DefaultPollConfig returns the default polling configuration:
// 2s initial interval, 15s max interval, 3-minute timeout.
func DefaultPollConfig() PollConfig {
	return PollConfig{
		InitialInterval: 2 * time.Second,
		MaxInterval:     15 * time.Second,
		Timeout:         3 * time.Minute,
	}
}

// ProgressFunc is an optional callback invoked with status messages during
// polling. Callers can use it to print progress to users.
type ProgressFunc func(msg string)

// AwaitWorkflowCompletion polls for a workflow run created after dispatchTime
// and waits for it to complete, using exponential backoff. It returns the
// completed run or an error if the context is cancelled or the timeout expires.
//
// The onProgress callback, if non-nil, is called with status messages during
// polling (e.g., "waiting for workflow run" or intermediate run status).
func AwaitWorkflowCompletion(
	ctx context.Context,
	client forge.Client,
	org, repo, workflowFile string,
	dispatchTime time.Time,
	cfg PollConfig,
	onProgress ProgressFunc,
) (*forge.WorkflowRun, error) {
	if cfg.InitialInterval <= 0 {
		return nil, fmt.Errorf("PollConfig.InitialInterval must be positive, got %v", cfg.InitialInterval)
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("PollConfig.Timeout must be positive, got %v", cfg.Timeout)
	}

	deadline := time.NewTimer(cfg.Timeout)
	defer deadline.Stop()

	interval := cfg.InitialInterval

	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("timed out waiting for %s workflow", workflowFile)
		case <-time.After(interval):
		}

		runs, err := client.ListWorkflowRuns(ctx, org, repo, workflowFile)
		if err != nil {
			if onProgress != nil {
				onProgress(fmt.Sprintf("waiting for workflow run (attempt %d)...", attempt+1))
			}
			interval = nextInterval(interval, cfg.MaxInterval)
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
			if onProgress != nil {
				onProgress(fmt.Sprintf("workflow run: %s (%s)", run.HTMLURL, run.Status))
			}
			break // found our run, keep waiting
		}

		interval = nextInterval(interval, cfg.MaxInterval)
	}
}

// nextInterval doubles the current interval, capping at max.
func nextInterval(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}
