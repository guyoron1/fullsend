package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
	"github.com/fullsend-ai/fullsend/internal/prescript"
	"github.com/fullsend-ai/fullsend/internal/ui"
)

// writePreScript creates an executable script for runPreScript tests.
func writePreScript(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("pre-script tests require a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), "pre-test.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/bash\nset -euo pipefail\n"+body), 0o755))
	return path
}

func TestRunPreScript_NoOutput_Proceeds(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t, "true\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.False(t, res.Skipped)
}

func TestRunPreScript_SkipRequested(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "reason=open PR exists" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.True(t, res.Skipped)
	assert.Equal(t, "open PR exists", res.Reason)
}

func TestRunPreScript_RunnerEnvVisible(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{
		PreScript: writePreScript(t,
			`[ "${MY_RUNNER_VAR}" = "on" ] || exit 7`+"\n"+
				`echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"),
		RunnerEnv: map[string]string{"MY_RUNNER_VAR": "on"},
	}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.True(t, res.Skipped)
}

func TestRunPreScript_ScriptFailureIsHardError(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t, "exit 3\n")}

	_, err := runPreScript(h, t.TempDir(), "", printer)
	require.ErrorContains(t, err, "running pre-script")
}

func TestRunPreScript_MalformedOutputIsHardError(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "skipped true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	_, err := runPreScript(h, t.TempDir(), "", printer)
	require.ErrorContains(t, err, "parsing pre-script output")
}

// The headline claim of issue #4718: a skip exits before the sandbox is
// ever created. usePreScriptStub makes sandbox creation fail loudly, so a
// nil error here proves runAgent returned first. If the pre-script block
// is ever moved below sandbox creation, this fails with "creating
// sandbox" — the error its paired no-skip test asserts on.
func TestRunAgent_PreScriptSkip_ReturnsBeforeSandboxCreation(t *testing.T) {
	usePreScriptStub(t)
	dir := newSkipHarnessDir(t, `echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
		`echo "reason=open PR exists" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	require.NoError(t, err)
}

// Without a skip, the run must still reach sandbox creation — a guard
// against the skip path swallowing every run — and skipped=false must be
// relayed so an absent key means only "this CLI predates the protocol".
// The two assertions share one run: reaching sandbox creation costs the
// full create-retry backoff.
func TestRunAgent_PreScriptNoSkip_ProceedsToSandboxAndRelaysFalse(t *testing.T) {
	usePreScriptStub(t)
	out := filepath.Join(t.TempDir(), "github-output")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_OUTPUT", out)
	dir := newSkipHarnessDir(t, "true\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating sandbox")

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "skipped=false\n", string(data))
}

// A harness with no pre_script must still relay skipped=false, otherwise
// an empty output would mean two different things and the documented
// three-state contract would not hold.
func TestRunAgent_NoPreScript_StillRelaysSkippedFalse(t *testing.T) {
	usePreScriptStub(t)
	out := filepath.Join(t.TempDir(), "github-output")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_OUTPUT", out)
	dir := newSkipHarnessDir(t, "")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating sandbox")

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "skipped=false\n", string(data))
}

// The skip path relays skipped=true. Fast: it returns before sandbox
// creation, so it does not pay the create-retry backoff.
func TestRunAgent_PreScriptSkip_RelaysSkippedTrue(t *testing.T) {
	usePreScriptStub(t)
	out := filepath.Join(t.TempDir(), "github-output")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_OUTPUT", out)
	dir := newSkipHarnessDir(t, `echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
		`echo "reason=open PR exists" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	require.NoError(t, runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "",
		rFlags, statusOpts{}, ui.New(io.Discard), false))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "skipped=true\nreason=open PR exists\n", string(data))
}

// A relay target that cannot be written must fail the run rather than
// exiting 0 with a decision the workflow gate never sees.
func TestRunAgent_PreScriptRelayFailureIsHardError(t *testing.T) {
	usePreScriptStub(t)
	t.Setenv("GITHUB_ACTIONS", "true")
	// A directory can be opened but not written to.
	t.Setenv("GITHUB_OUTPUT", t.TempDir())
	dir := newSkipHarnessDir(t, `echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	require.ErrorContains(t, err, "relaying pre-script outputs")
}

func TestRunPreScript_OutputFileExistsAndIsWritable(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`[ -f "${FULLSEND_PRESCRIPT_OUTPUT}" ] || exit 8`+"\n"+
			`[ -w "${FULLSEND_PRESCRIPT_OUTPUT}" ] || exit 9`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.False(t, res.Skipped)
}

// The output file is removed once parsed, so skips do not accumulate
// files in the run directory.
func TestRunPreScript_CleansUpOutputFile(t *testing.T) {
	printer := ui.New(io.Discard)
	runDir := t.TempDir()
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "skipped=true" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	_, err := runPreScript(h, runDir, "", printer)
	require.NoError(t, err)

	entries, err := os.ReadDir(runDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// Pre-script outputs (other than skipped/reason) are injected into the
// sandbox environment so the agent can consume computed values (#791).
// This test verifies the outputs appear in h.Env.Sandbox via
// buildSandboxEnvLines after runPreScript and the injection step.
func TestRunPreScript_OutputsFlowIntoSandboxEnv(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "COMPUTED_TOKEN=abc123" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "TARGET_URL=https://example.com" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "reason=just testing" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.False(t, res.Skipped)

	// Simulate the injection step from runAgent (step 4-post).
	sandboxOutputs := prescript.SandboxEnv(res)
	require.Len(t, sandboxOutputs, 2)
	assert.Equal(t, "abc123", sandboxOutputs["COMPUTED_TOKEN"])
	assert.Equal(t, "https://example.com", sandboxOutputs["TARGET_URL"])
	assert.NotContains(t, sandboxOutputs, "reason")

	// Merge into harness env and verify buildSandboxEnvLines picks them up.
	if h.Env == nil {
		h.Env = &harness.EnvConfig{}
	}
	if h.Env.Sandbox == nil {
		h.Env.Sandbox = make(map[string]string)
	}
	for k, v := range sandboxOutputs {
		h.Env.Sandbox[k] = v
	}
	lines := buildSandboxEnvLines(h)
	require.Len(t, lines, 2)
	assert.Contains(t, lines, "export COMPUTED_TOKEN='abc123'")
	assert.Contains(t, lines, "export TARGET_URL='https://example.com'")
}

// Pre-script outputs override static env.sandbox entries: the pre-script
// ran after env.sandbox was expanded, so its values are computed from
// runtime context that the static config cannot know.
func TestRunPreScript_OutputsOverrideStaticEnvSandbox(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{
		PreScript: writePreScript(t,
			`echo "TOKEN=dynamic_value" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"),
		Env: &harness.EnvConfig{
			Sandbox: map[string]string{"TOKEN": "static_value", "OTHER": "kept"},
		},
	}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)

	sandboxOutputs := prescript.SandboxEnv(res)
	for k, v := range sandboxOutputs {
		h.Env.Sandbox[k] = v
	}

	lines := buildSandboxEnvLines(h)
	assert.Contains(t, lines, "export TOKEN='dynamic_value'")
	assert.Contains(t, lines, "export OTHER='kept'")
}

// Pre-script outputs that use hyphenated keys (valid in prescript protocol)
// are skipped by buildSandboxEnvLines because they are not valid POSIX
// env var names.
func TestRunPreScript_HyphenatedKeysSkippedInSandboxEnv(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "existing-pr=123" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "VALID_KEY=ok" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)

	// SandboxEnv filters out hyphenated keys (not valid POSIX identifiers)
	// so the injected count matches what buildSandboxEnvLines will export.
	sandboxOutputs := prescript.SandboxEnv(res)
	require.Len(t, sandboxOutputs, 1)
	assert.Equal(t, "ok", sandboxOutputs["VALID_KEY"])
	assert.NotContains(t, sandboxOutputs, "existing-pr")

	if h.Env == nil {
		h.Env = &harness.EnvConfig{}
	}
	h.Env.Sandbox = make(map[string]string)
	for k, v := range sandboxOutputs {
		h.Env.Sandbox[k] = v
	}

	lines := buildSandboxEnvLines(h)
	require.Len(t, lines, 1)
	assert.Equal(t, "export VALID_KEY='ok'", lines[0])
}

// No pre-script outputs means no sandbox env changes (#791).
func TestRunPreScript_NoOutputs_NoSandboxEnvChanges(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t, "true\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)

	sandboxOutputs := prescript.SandboxEnv(res)
	assert.Nil(t, sandboxOutputs)
}

// Pre-script sandbox env injection via runAgent: when a pre-script emits
// non-reserved outputs, runAgent must reach sandbox creation with those
// outputs merged into h.Env.Sandbox. The openshell stub rejects sandbox
// creation, so the error message proves we reached that point — and
// buildSandboxEnvLines in bootstrapEnv would produce the exports.
func TestRunAgent_PreScriptOutputs_ReachSandboxCreation(t *testing.T) {
	usePreScriptStub(t)
	dir := newSkipHarnessDir(t,
		`echo "MY_COMPUTED=hello" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	// If we reached sandbox creation, the pre-script outputs were merged
	// into env.sandbox and the flow continued past the injection step.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating sandbox")
}

// usePreScriptStub puts an openshell stub on PATH that passes the gateway
// check but refuses sandbox creation, so a run that gets that far fails
// recognizably.
func usePreScriptStub(t *testing.T) {
	t.Helper()
	stubDir, err := filepath.Abs(filepath.Join("testdata", "prescript-stub"))
	require.NoError(t, err)
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))
}

// newSkipHarnessDir builds a minimal fullsend dir whose code harness runs
// the given pre-script body.
func newSkipHarnessDir(t *testing.T, preScriptBody string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "harness"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agents", "code.md"),
		[]byte("You are a coding agent."), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"),
		[]byte("agents:\n  - harness/code.yaml\n"), 0o644))

	harnessYAML := "agent: agents/code.md\nrole: test\n"
	if preScriptBody != "" {
		harnessYAML += "pre_script: " + writePreScript(t, preScriptBody) + "\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "harness", "code.yaml"),
		[]byte(harnessYAML), 0o644))
	return dir
}
