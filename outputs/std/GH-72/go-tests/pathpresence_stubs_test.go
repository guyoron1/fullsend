package scaffold

// STD Test Stubs for GH-72: ComparePathPresence batch path checking
// Suite: TS-GH72-001
//
// These stubs correspond to test cases TC-GH72-001 through TC-GH72-006.
// Production tests: internal/scaffold/pathpresence_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-001: All expected paths are present in repository
//
// Preconditions:
//   - FakeClient populated with FileContents matching 3 expected paths
//     (action.yml, reusable-triage.yml, bin/fullsend) under org/.fullsend/
//
// Steps:
//  1. Call ComparePathPresence with the same 3 paths as expected
//
// Expected:
//   - Returns nil error and empty missing slice
func TestComparePathPresence_AllPresent_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-001")
}

// TC-GH72-002: Some expected paths are missing from repository
//
// Preconditions:
//   - FakeClient has action.yml and bin/fullsend but NOT reusable-triage.yml
//     and reusable-code.yml
//
// Steps:
//  1. Call ComparePathPresence with 4 expected paths (2 present, 2 missing)
//
// Expected:
//   - Returns sorted slice of 2 missing paths:
//     [".github/workflows/reusable-code.yml", ".github/workflows/reusable-triage.yml"]
func TestComparePathPresence_SomeMissing_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-002")
}

// TC-GH72-003: All expected paths are missing from empty repository
//
// Preconditions:
//   - FakeClient has empty FileContents map (no files in repo)
//
// Steps:
//  1. Call ComparePathPresence with 2 expected paths
//
// Expected:
//   - Returns both paths in sorted missing slice
func TestComparePathPresence_AllMissing_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-003")
}

// TC-GH72-004: Empty expected list returns no missing paths
//
// Preconditions:
//   - FakeClient may have file contents (irrelevant — function short-circuits)
//
// Steps:
//  1. Call ComparePathPresence with nil expected slice
//
// Expected:
//   - Returns nil error and nil missing slice
//   - No API call to ListRepositoryFiles is made
func TestComparePathPresence_EmptyExpected_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-004")
}

// TC-GH72-005: Forge client error is propagated
//
// Preconditions:
//   - FakeClient has ListRepositoryFiles error injected ("network error")
//
// Steps:
//  1. Call ComparePathPresence with one expected path
//
// Expected:
//   - Returns error wrapping the original, containing "listing repository files"
func TestComparePathPresence_ForgeError_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-005")
}

// TC-GH72-006: Uses single batch API call instead of per-path GetFileContent
//
// Preconditions:
//   - FakeClient has 2 files (path-a, path-b) in FileContents
//   - GetFileContent error injected ("should not be called") as a trip-wire
//
// Steps:
//  1. Call ComparePathPresence with 3 paths (path-a, path-b, path-c)
//
// Expected:
//   - Returns no error (GetFileContent trip-wire not triggered)
//   - Missing list contains only ["path-c"]
//   - Proves ListRepositoryFiles (O(1) batch) is used instead of GetFileContent (O(N))
func TestComparePathPresence_UsesOneAPICall_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-006")
}
