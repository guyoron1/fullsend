package harness

// STD Test Stubs for GH-72: DiscoverRemoteAgents harness discovery via forge API
// Suite: TS-GH72-005
//
// These stubs correspond to test cases TC-GH72-025 through TC-GH72-036.
// Production tests: internal/harness/discover_remote_test.go
// STP reference: outputs/stp/GH-72/GH-72_test_plan.md

import "testing"

// TC-GH72-025: Multiple harnesses discovered and sorted by role
//
// Preconditions:
//   - FakeClient has 3 harness YAML files (triage.yaml, code.yaml, review.yaml)
//     in DirContents for harness/ directory
//   - Each file has valid YAML with role and slug fields
//
// Steps:
//  1. Call DiscoverRemoteAgents(ctx, client, "acme", ".fullsend", "main")
//
// Expected:
//   - Returns 3 agents sorted alphabetically by role: coder, review, triage
//   - Each agent has correct role, slug, and filename
func TestDiscoverRemoteAgents_MultipleSorted_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-025")
}

// TC-GH72-026: Missing harness directory returns nil,nil
//
// Preconditions:
//   - FakeClient has no DirContents entry for harness/ (directory does not exist)
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns (nil, nil) — not-found is not an error
func TestDiscoverRemoteAgents_NoHarnessDir_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-026")
}

// TC-GH72-027: Files without role or slug are skipped
//
// Preconditions:
//   - FakeClient has legacy.yaml (no role/slug fields) and modern.yaml (has both)
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns 1 agent (modern.yaml only)
//   - legacy.yaml excluded from results
func TestDiscoverRemoteAgents_SkipsNoRoleNoSlug_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-027")
}

// TC-GH72-028: Malformed YAML returns partial results with multi-error
//
// Preconditions:
//   - FakeClient has good.yaml (valid) and bad.yaml (invalid YAML syntax)
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns 1 agent (good.yaml) AND error containing "bad.yaml"
//   - Valid files returned despite per-file errors
func TestDiscoverRemoteAgents_MalformedYAML_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-028")
}

// TC-GH72-029: Subdirectories are skipped
//
// Preconditions:
//   - DirContents has triage.yaml (type="file") and subdir (type="dir")
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns 1 agent (triage.yaml only)
//   - Subdirectory entry ignored
func TestDiscoverRemoteAgents_SkipsSubdirs_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-029")
}

// TC-GH72-030: ListDirectoryContents error propagates
//
// Preconditions:
//   - FakeClient has ListDirectoryContents error injected ("network error")
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Error returned containing "listing harness directory"
//   - agents is nil
func TestDiscoverRemoteAgents_ListDirError_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-030")
}

// TC-GH72-031: Same role sorted by filename
//
// Preconditions:
//   - FakeClient has fix.yaml and code.yaml, both with role="coder"
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns 2 agents: code.yaml before fix.yaml (alphabetical by filename)
func TestDiscoverRemoteAgents_SameRoleSortedByFilename_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-031")
}

// TC-GH72-032: Role-only file (no slug) is included
//
// Preconditions:
//   - YAML file has role="triage" but no slug field
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Agent returned with role="triage", Slug="" (empty)
func TestDiscoverRemoteAgents_RoleOnly_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-032")
}

// TC-GH72-033: Slug-only file (no role) is included
//
// Preconditions:
//   - YAML file has slug="fs-triage" but no role field
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Agent returned with slug="fs-triage", Role="" (empty)
func TestDiscoverRemoteAgents_SlugOnly_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-033")
}

// TC-GH72-034: .yml extension files are discovered
//
// Preconditions:
//   - DirContents has agent.yml (not .yaml)
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - agent.yml is parsed and included in results
//   - Filename in result is "agent.yml"
func TestDiscoverRemoteAgents_YmlExtension_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-034")
}

// TC-GH72-035: Empty harness directory returns empty list
//
// Preconditions:
//   - DirContents has entry for harness/ with empty entries list
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - Returns empty slice (not nil) and nil error
func TestDiscoverRemoteAgents_EmptyDir_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-035")
}

// TC-GH72-036: Path field is empty for remote agents
//
// Preconditions:
//   - Valid remote harness file with role and slug
//
// Steps:
//  1. Call DiscoverRemoteAgents
//
// Expected:
//   - AgentInfo.Path is empty string
//   - Only local discovery (DiscoverAgents) populates the Path field
func TestDiscoverRemoteAgents_PathEmpty_Stub(t *testing.T) {
	t.Skip("stub: TC-GH72-036")
}
