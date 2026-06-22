package scaffold

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-001 — ComparePathPresence batch path checking
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/forge"
)

// TC-GH72-001: All expected paths are present in repository
func TestQFComparePathPresence_AllPresent(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org/.fullsend/.defaults/action.yml":                  []byte("marker"),
			"org/.fullsend/.github/workflows/reusable-triage.yml": []byte("wf"),
			"org/.fullsend/bin/fullsend":                          []byte("binary"),
		},
	}

	missing, err := ComparePathPresence(context.Background(), client, "org", ".fullsend", []string{
		".defaults/action.yml",
		".github/workflows/reusable-triage.yml",
		"bin/fullsend",
	})
	require.NoError(t, err)
	assert.Empty(t, missing, "all expected paths exist, missing should be empty")
}

// TC-GH72-002: Some expected paths are missing from repository
func TestQFComparePathPresence_SomeMissing(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org/.fullsend/.defaults/action.yml": []byte("marker"),
			"org/.fullsend/bin/fullsend":         []byte("binary"),
		},
	}

	missing, err := ComparePathPresence(context.Background(), client, "org", ".fullsend", []string{
		".defaults/action.yml",
		".github/workflows/reusable-triage.yml",
		".github/workflows/reusable-code.yml",
		"bin/fullsend",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		".github/workflows/reusable-code.yml",
		".github/workflows/reusable-triage.yml",
	}, missing, "missing paths should be returned in sorted order")
}

// TC-GH72-003: All expected paths are missing from empty repository
func TestQFComparePathPresence_AllMissing(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{},
	}

	missing, err := ComparePathPresence(context.Background(), client, "org", ".fullsend", []string{
		".defaults/action.yml",
		"bin/fullsend",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{".defaults/action.yml", "bin/fullsend"}, missing,
		"all expected paths should appear in missing list")
}

// TC-GH72-004: Empty expected list returns no missing paths
func TestQFComparePathPresence_EmptyExpected(t *testing.T) {
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org/.fullsend/bin/fullsend": []byte("binary"),
		},
	}

	missing, err := ComparePathPresence(context.Background(), client, "org", ".fullsend", nil)
	require.NoError(t, err)
	assert.Nil(t, missing, "nil expected slice should return nil missing slice without API call")
}

// TC-GH72-005: Forge client error is propagated
func TestQFComparePathPresence_ForgeError(t *testing.T) {
	client := &forge.FakeClient{
		Errors: map[string]error{
			"ListRepositoryFiles": errors.New("network error"),
		},
	}

	_, err := ComparePathPresence(context.Background(), client, "org", ".fullsend", []string{
		".defaults/action.yml",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing repository files",
		"error should wrap the original forge client error")
}

// TC-GH72-006: Uses single batch API call instead of per-path GetFileContent
func TestQFComparePathPresence_UsesOneAPICall(t *testing.T) {
	// Inject GetFileContent error as a trip-wire to prove it is never called.
	// ComparePathPresence must use ListRepositoryFiles exclusively.
	client := &forge.FakeClient{
		FileContents: map[string][]byte{
			"org/repo/path-a": []byte("a"),
			"org/repo/path-b": []byte("b"),
		},
		Errors: map[string]error{
			"GetFileContent": errors.New("should not be called"),
		},
	}

	missing, err := ComparePathPresence(context.Background(), client, "org", "repo", []string{
		"path-a",
		"path-b",
		"path-c",
	})
	require.NoError(t, err, "GetFileContent should never be called — only ListRepositoryFiles")
	assert.Equal(t, []string{"path-c"}, missing)
}
