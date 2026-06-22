package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Section 3.6 — Post-Review — Diff Hunk Parsing
// =============================================================================

// TS-GH73-044: Single hunk @@ -10,5 +12,7 @@ — range [12,18]
func TestQF_ParseDiffLineRanges_SingleHunk(t *testing.T) {
	patch := "@@ -10,5 +12,7 @@ func foo() {\n some code\n more code"

	ranges := parseDiffLineRanges(patch)

	assert.Len(t, ranges, 1)
	assert.Equal(t, 12, ranges[0][0]) // Start
	assert.Equal(t, 18, ranges[0][1]) // End = 12 + 7 - 1
}

// TS-GH73-045: Multiple hunks — multiple ranges
func TestQF_ParseDiffLineRanges_MultipleHunks(t *testing.T) {
	patch := "@@ -10,5 +12,7 @@ func foo() {\n code\n@@ -30,3 +40,5 @@ func bar() {\n code"

	ranges := parseDiffLineRanges(patch)

	assert.Len(t, ranges, 2)
	assert.Equal(t, 12, ranges[0][0])
	assert.Equal(t, 18, ranges[0][1])
	assert.Equal(t, 40, ranges[1][0])
	assert.Equal(t, 44, ranges[1][1])
}

// TS-GH73-046: New file @@ -0,0 +1,50 @@ — range [1,50]
func TestQF_ParseDiffLineRanges_NewFile(t *testing.T) {
	patch := "@@ -0,0 +1,50 @@\n+package main"

	ranges := parseDiffLineRanges(patch)

	assert.Len(t, ranges, 1)
	assert.Equal(t, 1, ranges[0][0])
	assert.Equal(t, 50, ranges[0][1])
}

// TS-GH73-047: Deletion-only hunk — no range emitted
func TestQF_ParseDiffLineRanges_DeletionOnly(t *testing.T) {
	patch := "@@ -10,5 +10,0 @@\n-deleted line 1\n-deleted line 2"

	ranges := parseDiffLineRanges(patch)

	assert.Empty(t, ranges)
}

// TS-GH73-048: Omitted size defaults to 1
func TestQF_ParseDiffLineRanges_OmittedSize(t *testing.T) {
	patch := "@@ -10 +12 @@\n some code"

	ranges := parseDiffLineRanges(patch)

	assert.Len(t, ranges, 1)
	assert.Equal(t, 12, ranges[0][0])
	assert.Equal(t, 12, ranges[0][1]) // Size=1 → End = Start
}

// TS-GH73-049: Empty patch — nil ranges
func TestQF_ParseDiffLineRanges_EmptyPatch(t *testing.T) {
	patch := ""

	ranges := parseDiffLineRanges(patch)

	assert.Nil(t, ranges)
}
