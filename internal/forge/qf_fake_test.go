package forge

// QualityFlow generated tests for GH-72
// Suite: TS-GH72-002 — FakeClient ListRepositoryFiles implementation
// STD: outputs/std/GH-72/GH-72_test_description.yaml

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TC-GH72-007: FakeClient ListRepositoryFiles error injection
func TestQFFakeClient_ErrorInjection_ListRepositoryFiles(t *testing.T) {
	ctx := context.Background()
	injected := errors.New("injected error")

	fc := &FakeClient{
		Errors: map[string]error{
			"ListRepositoryFiles": injected,
		},
	}

	_, err := fc.ListRepositoryFiles(ctx, "o", "r")
	require.Error(t, err)
	assert.ErrorIs(t, err, injected, "injected error should be returned via errors.Is")
}

// TC-GH72-008: FakeClient thread safety for ListRepositoryFiles
func TestQFFakeClient_ThreadSafety_ListRepositoryFiles(t *testing.T) {
	ctx := context.Background()
	fc := &FakeClient{
		FileContents: map[string][]byte{
			"o/r/file1.txt": []byte("content1"),
			"o/r/file2.txt": []byte("content2"),
		},
	}

	var wg sync.WaitGroup
	const goroutines = 20

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			files, err := fc.ListRepositoryFiles(ctx, "o", "r")
			assert.NoError(t, err)
			assert.Len(t, files, 2, "should return 2 files from concurrent access")
		}()
	}

	wg.Wait()
}
