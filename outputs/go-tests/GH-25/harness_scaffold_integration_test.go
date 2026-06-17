//go:build e2e

package harness_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fullsend-ai/fullsend/internal/harness"
)

/*
Harness Scaffold Integration & parseRaw Tests

STP Reference: outputs/stp/GH-25/GH-25_test_plan.md
Jira: GH-25
*/

func TestScaffoldIntegration(t *testing.T) {
	// [test_id:TS-GH-25-049] should validate generated harness files against schema
	t.Run("[test_id:TS-GH-25-049] should validate generated harness files against schema", func(t *testing.T) {
		// Create a well-formed harness file that represents what the scaffold
		// generator would produce, and verify it passes Validate().
		tmpDir := t.TempDir()
		harnessContent := []byte(`agent: claude
role: triage
slug: triage-agent
description: "Triage agent for issue classification"
model: sonnet
`)
		harnessPath := filepath.Join(tmpDir, "triage.yaml")
		require.NoError(t, os.WriteFile(harnessPath, harnessContent, 0644))

		h, err := harness.Load(harnessPath)

		require.NoError(t, err, "well-formed harness file should load and validate")
		require.NotNil(t, h)
		assert.Equal(t, "triage", h.Role)
		assert.Equal(t, "triage-agent", h.Slug)
	})
}

func TestParseRaw(t *testing.T) {
	// parseRaw is unexported, so we test its behavior through LoadRaw which
	// reads from file and calls parseRaw internally.

	// [test_id:TS-GH-25-050] should parse valid YAML bytes into Harness struct
	t.Run("[test_id:TS-GH-25-050] should parse valid YAML bytes into Harness struct", func(t *testing.T) {
		tmpDir := t.TempDir()
		validYAML := []byte(`role: triage
slug: triage-agent
description: "Agent for triage"
model: sonnet
`)
		yamlPath := filepath.Join(tmpDir, "valid.yaml")
		require.NoError(t, os.WriteFile(yamlPath, validYAML, 0644))

		h, err := harness.LoadRaw(yamlPath)

		require.NoError(t, err, "valid YAML should parse without error")
		require.NotNil(t, h)
		assert.Equal(t, "triage", h.Role)
		assert.Equal(t, "triage-agent", h.Slug)
	})

	// [test_id:TS-GH-25-051] should return parse error for invalid YAML
	t.Run("[test_id:TS-GH-25-051] should return parse error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidYAML := []byte(":::invalid yaml{{{")
		yamlPath := filepath.Join(tmpDir, "bad.yaml")
		require.NoError(t, os.WriteFile(yamlPath, invalidYAML, 0644))

		h, err := harness.LoadRaw(yamlPath)

		require.Error(t, err, "invalid YAML should return an error")
		assert.Nil(t, h, "harness should be nil on parse error")
	})
}
