package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GH-73-TC-038: Verify workflow YAML renders correctly
func TestQF_RenderTemplate_SubstitutesVars(t *testing.T) {
	t.Run("vendored per-org replaces workflow placeholder", func(t *testing.T) {
		raw, err := FullsendRepoFile(".github/workflows/triage.yml")
		require.NoError(t, err)

		rendered, err := RenderTemplate(".github/workflows/triage.yml", raw, RenderOptions{
			Vendored: true,
			PerRepo:  false,
		})
		require.NoError(t, err)
		out := string(rendered)
		assert.Contains(t, out, "uses: ./.github/workflows/reusable-triage.yml")
		assert.NotContains(t, out, "__REUSABLE_WORKFLOW__", "placeholder should be fully substituted")
	})

	t.Run("vendored per-repo replaces workflow placeholder", func(t *testing.T) {
		raw, err := FullsendRepoFile(".github/workflows/triage.yml")
		require.NoError(t, err)

		rendered, err := RenderTemplate(".github/workflows/triage.yml", raw, RenderOptions{
			Vendored: true,
			PerRepo:  true,
		})
		require.NoError(t, err)
		out := string(rendered)
		assert.Contains(t, out, "uses: ./.fullsend/.github/workflows/reusable-triage.yml")
	})

	t.Run("not vendored uses upstream repo reference", func(t *testing.T) {
		raw, err := FullsendRepoFile(".github/workflows/triage.yml")
		require.NoError(t, err)

		rendered, err := RenderTemplate(".github/workflows/triage.yml", raw, RenderOptions{
			Vendored: false,
		})
		require.NoError(t, err)
		out := string(rendered)
		assert.Contains(t, out, "fullsend-ai/fullsend/.github/workflows/reusable-triage.yml@")
	})
}

// GH-73-TC-038 supplemental: Verify per-repo dispatch template renders
func TestQF_RenderTemplate_PerRepoDispatch(t *testing.T) {
	raw, err := PerRepoShimTemplate()
	require.NoError(t, err)

	rendered, err := RenderTemplate("templates/shim-per-repo.yaml", raw, RenderOptions{
		Vendored: true,
		PerRepo:  true,
	})
	require.NoError(t, err)
	out := string(rendered)
	assert.Contains(t, out, "uses: ./.fullsend/.github/workflows/reusable-dispatch.yml")
	assert.NotContains(t, out, "__REUSABLE_DISPATCH__", "placeholder should be substituted")
}

// GH-73-TC-038 supplemental: RenderOptionsForInstall builds correct options
func TestQF_RenderOptionsForInstall(t *testing.T) {
	opts := RenderOptionsForInstall(true, true)
	assert.True(t, opts.Vendored)
	assert.True(t, opts.PerRepo)

	opts = RenderOptionsForInstall(false, false)
	assert.False(t, opts.Vendored)
	assert.False(t, opts.PerRepo)
}
