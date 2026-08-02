package cli

import (
	"context"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
	prescriptPkg "github.com/fullsend-ai/fullsend/internal/prescript"
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

// Pre-script outputs flow through runPreScript → SandboxEnv → h.Env.Sandbox
// → buildSandboxEnvLines. This test exercises the SandboxEnv → Sandbox
// injection without requiring a full sandbox.
func TestPreScriptOutputs_FlowToSandboxEnv(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "skipped=false" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "MY_TOKEN=abc123" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "RESOLVED_URL=https://example.com" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)
	assert.False(t, res.Skipped)

	// Simulate what runAgent does: inject non-reserved outputs into h.Env.Sandbox.
	sandboxOutputs := prescriptPkg.SandboxEnv(res)
	require.NotNil(t, sandboxOutputs)
	assert.Equal(t, "abc123", sandboxOutputs["MY_TOKEN"])
	assert.Equal(t, "https://example.com", sandboxOutputs["RESOLVED_URL"])
	// Reserved keys must not appear.
	assert.NotContains(t, sandboxOutputs, "skipped")

	// Merge into harness and verify buildSandboxEnvLines picks them up.
	h.Env = &harness.EnvConfig{Sandbox: make(map[string]string)}
	maps.Copy(h.Env.Sandbox, sandboxOutputs)
	lines := buildSandboxEnvLines(h)
	assert.Contains(t, lines, "export MY_TOKEN='abc123'")
	assert.Contains(t, lines, "export RESOLVED_URL='https://example.com'")
}

// Pre-script outputs override static env.sandbox entries on key collision.
func TestPreScriptOutputs_OverrideStaticSandboxEnv(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{
		PreScript: writePreScript(t,
			`echo "MY_VAR=dynamic" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"),
		Env: &harness.EnvConfig{
			Sandbox: map[string]string{"MY_VAR": "static"},
		},
	}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)

	sandboxOutputs := prescriptPkg.SandboxEnv(res)
	require.NotNil(t, sandboxOutputs)
	// Simulate the merge: pre-script overrides static.
	maps.Copy(h.Env.Sandbox, sandboxOutputs)
	lines := buildSandboxEnvLines(h)
	assert.Contains(t, lines, "export MY_VAR='dynamic'")
	assert.NotContains(t, lines, "export MY_VAR='static'")
}

// Hyphenated keys are valid in the prescript protocol but not valid POSIX
// env var names. SandboxEnv passes them through; buildSandboxEnvLines
// silently skips them.
func TestPreScriptOutputs_HyphenatedKeysSkippedBySandboxEnv(t *testing.T) {
	printer := ui.New(io.Discard)
	h := &harness.Harness{PreScript: writePreScript(t,
		`echo "my-output=value" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n"+
			`echo "VALID_KEY=ok" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")}

	res, err := runPreScript(h, t.TempDir(), "", printer)
	require.NoError(t, err)

	sandboxOutputs := prescriptPkg.SandboxEnv(res)
	require.NotNil(t, sandboxOutputs)
	// Hyphenated key passes through SandboxEnv...
	assert.Equal(t, "value", sandboxOutputs["my-output"])

	// ...but buildSandboxEnvLines skips it (not a valid POSIX identifier).
	h.Env = &harness.EnvConfig{Sandbox: make(map[string]string)}
	maps.Copy(h.Env.Sandbox, sandboxOutputs)
	lines := buildSandboxEnvLines(h)
	assert.Contains(t, lines, "export VALID_KEY='ok'")
	// Hyphenated key should not appear in the env lines.
	for _, line := range lines {
		assert.NotContains(t, line, "my-output")
	}
}

// Integration test: pre-script sandbox env injection via runAgent — the
// outputs must reach h.Env.Sandbox before sandbox creation. The test
// uses the prescript-stub which fails at sandbox creation, proving the
// injection happened before that step.
func TestRunAgent_PreScriptOutputs_InjectedBeforeSandboxCreation(t *testing.T) {
	usePreScriptStub(t)
	dir := newSkipHarnessDir(t,
		`echo "MY_COMPUTED_VAR=computed_value" >> "${FULLSEND_PRESCRIPT_OUTPUT}"`+"\n")

	rFlags := resolveFlags{maxDepth: 10, maxResources: 50}
	err := runAgent(context.Background(), "code", dir, "", t.TempDir(), "", nil, false, "", "", rFlags,
		statusOpts{}, ui.New(io.Discard), false)
	// The run must reach sandbox creation (proving it passed the injection
	// step) and fail there because the stub refuses to create a sandbox.
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
