package layers

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

/*
Enrollment Layer Stack Integration Tests

STP Reference: outputs/stp/GH-2354/GH-2354_test_plan.md
STD Reference: outputs/std/GH-2354/GH-2354_test_description.yaml
Jira: GH-2354
Section: 4.6 Layer Stack Integration
*/

// fakeLayer is a minimal Layer implementation for testing stack behavior.
type fakeLayer struct {
	name      string
	installed bool
	installFn func(ctx context.Context) error
}

func (f *fakeLayer) Name() string                       { return f.name }
func (f *fakeLayer) RequiredScopes(_ Operation) []string { return nil }
func (f *fakeLayer) Install(ctx context.Context) error {
	f.installed = true
	if f.installFn != nil {
		return f.installFn(ctx)
	}
	return nil
}
func (f *fakeLayer) Uninstall(_ context.Context) error               { return nil }
func (f *fakeLayer) Analyze(_ context.Context) (*LayerReport, error) { return nil, nil }

func TestEnrollmentLayerStack(t *testing.T) {
	t.Run("[test_id:TS-GH2354-018] InstallAll continues after enrollment timeout", func(t *testing.T) {
		// Verify that when a layer returns nil (as enrollment does on timeout),
		// InstallAll continues to subsequent layers. We simulate this with a
		// fakeLayer that mimics enrollment's non-fatal timeout behavior,
		// because the real enrollment layer's 3-minute internal timeout is
		// too slow for tests, and using a short context timeout would expire
		// the shared context (affecting subsequent layers via ctx.Err() check).
		timeoutLayer := &fakeLayer{
			name: "enrollment",
			installFn: func(_ context.Context) error {
				// Simulate enrollment timeout: returns nil (non-fatal).
				return nil
			},
		}

		postEnroll := &fakeLayer{name: "post-enrollment"}
		stack := NewStack(timeoutLayer, postEnroll)

		err := stack.InstallAll(context.Background())

		require.NoError(t, err)
		assert.True(t, timeoutLayer.installed, "enrollment layer should have been called")
		assert.True(t, postEnroll.installed,
			"subsequent layer should execute after enrollment returns nil (non-fatal timeout)")
	})

	t.Run("[test_id:TS-GH2354-019] InstallAll stops on enrollment dispatch error", func(t *testing.T) {
		// When the enrollment layer returns a fatal error (dispatch failure),
		// InstallAll should stop and not execute subsequent layers.
		client := &forge.FakeClient{
			Errors: map[string]error{
				"DispatchWorkflow": assert.AnError,
			},
		}
		enrollLayer, _ := newEnrollmentLayer(t, client, []string{"repo-a"}, nil)

		postEnroll := &fakeLayer{name: "post-enrollment"}
		stack := NewStack(enrollLayer, postEnroll)

		err := stack.InstallAll(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "layer enrollment:")
		assert.False(t, postEnroll.installed,
			"subsequent layer should NOT execute after fatal enrollment error")
	})
}
