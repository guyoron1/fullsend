package cf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePreviewAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
		errMsg  string
	}{
		{"valid simple", "pr-123", false, ""},
		{"valid single char", "a", false, ""},
		{"valid long", "my-preview-alias-for-testing-123", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"uppercase", "PR-123", true, "invalid preview alias"},
		{"leading hyphen", "-pr-123", true, "invalid preview alias"},
		{"trailing hyphen", "pr-123-", true, "invalid preview alias"},
		{"underscore", "pr_123", true, "invalid preview alias"},
		{"dot", "pr.123", true, "invalid preview alias"},
		{"space", "pr 123", true, "invalid preview alias"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePreviewAlias(tc.alias)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateWorkerName(t *testing.T) {
	tests := []struct {
		name    string
		worker  string
		wantErr bool
		errMsg  string
	}{
		{"valid", "mint-test", false, ""},
		{"valid single char", "a", false, ""},
		{"empty", "", true, "cannot be empty"},
		{"uppercase", "MINT", true, "invalid worker name"},
		{"leading hyphen", "-mint", true, "invalid worker name"},
		{"trailing hyphen", "mint-", true, "invalid worker name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorkerName(tc.worker)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPreviewURL(t *testing.T) {
	url := PreviewURL("pr-42", "mint-test", "my-account")
	assert.Equal(t, "https://pr-42-mint-test.my-account.workers.dev", url)
}

func TestProductionURL(t *testing.T) {
	url := ProductionURL("mint-test", "my-account")
	assert.Equal(t, "https://mint-test.my-account.workers.dev", url)
}

func TestLiveRunner_PreviewTeardown_IsNoOp(t *testing.T) {
	r := NewLiveRunner()
	err := r.PreviewTeardown(context.Background(), "mint-test", "pr-42")
	require.NoError(t, err, "preview teardown should be a no-op")
}

// Compile-time check that LiveRunner implements Runner.
var _ Runner = (*LiveRunner)(nil)

func TestNewLiveRunner(t *testing.T) {
	r := NewLiveRunner()
	assert.Equal(t, "wrangler", r.WranglerBin)
}

func TestLiveRunner_WranglerBinDefault(t *testing.T) {
	r := &LiveRunner{}
	assert.Equal(t, "wrangler", r.wranglerBin())
}

func TestLiveRunner_WranglerBinCustom(t *testing.T) {
	r := &LiveRunner{WranglerBin: "/usr/local/bin/wrangler"}
	assert.Equal(t, "/usr/local/bin/wrangler", r.wranglerBin())
}

// stubRunner implements Runner for testing purposes.
type stubRunner struct {
	deployCalled        bool
	previewCalled       bool
	teardownCalled      bool
	previewTeardownCall bool
	lastWorker          string
	lastAlias           string
	lastSourceDir       string
}

func (s *stubRunner) Deploy(_ context.Context, workerName, sourceDir string) error {
	s.deployCalled = true
	s.lastWorker = workerName
	s.lastSourceDir = sourceDir
	return nil
}

func (s *stubRunner) PreviewUpload(_ context.Context, workerName, sourceDir, alias string) error {
	s.previewCalled = true
	s.lastWorker = workerName
	s.lastSourceDir = sourceDir
	s.lastAlias = alias
	return nil
}

func (s *stubRunner) Teardown(_ context.Context, workerName string) error {
	s.teardownCalled = true
	s.lastWorker = workerName
	return nil
}

func (s *stubRunner) PreviewTeardown(_ context.Context, workerName, alias string) error {
	s.previewTeardownCall = true
	s.lastWorker = workerName
	s.lastAlias = alias
	return nil
}

func TestStubRunner_Deploy(t *testing.T) {
	s := &stubRunner{}
	err := s.Deploy(context.Background(), "mint-test", "/tmp/src")
	require.NoError(t, err)
	assert.True(t, s.deployCalled)
	assert.Equal(t, "mint-test", s.lastWorker)
	assert.Equal(t, "/tmp/src", s.lastSourceDir)
}

func TestStubRunner_PreviewUpload(t *testing.T) {
	s := &stubRunner{}
	err := s.PreviewUpload(context.Background(), "mint-test", "/tmp/src", "pr-42")
	require.NoError(t, err)
	assert.True(t, s.previewCalled)
	assert.Equal(t, "mint-test", s.lastWorker)
	assert.Equal(t, "pr-42", s.lastAlias)
	assert.Equal(t, "/tmp/src", s.lastSourceDir)
}

func TestStubRunner_Teardown(t *testing.T) {
	s := &stubRunner{}
	err := s.Teardown(context.Background(), "mint-test")
	require.NoError(t, err)
	assert.True(t, s.teardownCalled)
	assert.Equal(t, "mint-test", s.lastWorker)
}

func TestStubRunner_PreviewTeardown(t *testing.T) {
	s := &stubRunner{}
	err := s.PreviewTeardown(context.Background(), "mint-test", "pr-42")
	require.NoError(t, err)
	assert.True(t, s.previewTeardownCall)
	assert.Equal(t, "mint-test", s.lastWorker)
	assert.Equal(t, "pr-42", s.lastAlias)
}
