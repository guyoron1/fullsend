package scaffold

/*
forge.Client Interface Compliance Tests

STP Reference: outputs/stp/GH-2351/GH-2351_test_plan.md
Jira: GH-2351
*/

import (
	"testing"

	"github.com/fullsend-ai/fullsend/internal/forge"
	"github.com/fullsend-ai/fullsend/internal/forge/github"
)

// Compile-time interface assertions — these fail at build time if either
// type does not implement forge.Client (including ListRepositoryFiles).
var (
	_ forge.Client = (*forge.FakeClient)(nil)
	_ forge.Client = (*github.LiveClient)(nil)
)

func TestInterfaceCompliance(t *testing.T) {
	t.Run("[test_id:TS-GH-2351-016] should verify FakeClient satisfies Client interface", func(t *testing.T) {
		// This is primarily a compile-time check (see var _ above).
		// If this test compiles and runs, FakeClient satisfies forge.Client.
		var _ forge.Client = (*forge.FakeClient)(nil)
	})

	t.Run("[test_id:TS-GH-2351-017] should verify LiveClient satisfies Client interface", func(t *testing.T) {
		// This is primarily a compile-time check (see var _ above).
		// If this test compiles and runs, LiveClient satisfies forge.Client.
		var _ forge.Client = (*github.LiveClient)(nil)
	})
}
