package forge

import (
	"context"
	"fmt"
	"time"
)

// AwaitForkReady polls GetDefaultBranch until the fork's default branch ref
// is readable, indicating that git data replication has completed. This is
// necessary after CreateForkInOrg because GitHub's fork API is asynchronous —
// it responds before git data is available.
//
// It polls up to 30 times with a 2-second interval (≈60 seconds total).
// Returns nil on success, or wraps the last error on timeout.
func AwaitForkReady(ctx context.Context, client Client, owner, repo string) error {
	const (
		maxAttempts = 30
		interval    = 2 * time.Second
	)

	var lastErr error
	for attempt := range maxAttempts {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("waiting for fork %s/%s: %w", owner, repo, ctx.Err())
			case <-time.After(interval):
			}
		}

		_, err := client.GetDefaultBranch(ctx, owner, repo)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("fork %s/%s not ready after %d attempts: %w",
		owner, repo, maxAttempts, lastErr)
}
