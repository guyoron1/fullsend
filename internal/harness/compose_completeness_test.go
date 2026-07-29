package harness

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// composeMergeExemptions maps "StructName.funcName" to the set of field names
// that are intentionally not handled by that merge function.
// Every exemption must have a non-empty justification.
//
// When adding a new field to a composable struct, either:
//  1. Update the merge function to handle it, OR
//  2. Add an entry here with a reason why carry-forward is not needed.
var composeMergeExemptions = map[string]map[string]string{
	"Harness.mergeBaseIntoChild": {
		"Base":                   "consumed and cleared by LoadWithBase after merge",
		"AllowedRemoteResources": "security: not merged to prevent privilege escalation (ADR-0045)",
		"AllowRuntimeFetch":      "security: not merged to prevent privilege escalation",
		"MaxRuntimeFetches":      "security: not merged to prevent privilege escalation",
	},
	"ForgeConfig.mergeForgeConfigInto": {},
	"ForgeConfig.mergeForgeConfig":     {},
	"ValidationLoop.mergeBaseIntoChild": {
		"Script":        "whole-struct pointer replace; individual fields not referenced",
		"MaxIterations": "whole-struct pointer replace; individual fields not referenced",
		"FeedbackMode":  "whole-struct pointer replace; individual fields not referenced",
	},
	"ValidationLoop.mergeForgeConfigInto": {
		"Script":        "whole-struct pointer replace; individual fields not referenced",
		"MaxIterations": "whole-struct pointer replace; individual fields not referenced",
		"FeedbackMode":  "whole-struct pointer replace; individual fields not referenced",
	},
	"ValidationLoop.mergeForgeConfig": {
		"Script":        "whole-struct pointer replace; individual fields not referenced",
		"MaxIterations": "whole-struct pointer replace; individual fields not referenced",
		"FeedbackMode":  "whole-struct pointer replace; individual fields not referenced",
	},
}

// TestComposeMergeCompleteness verifies that every field of each composable
// struct is either referenced in the relevant merge function body or
// explicitly exempted in composeMergeExemptions. This prevents a class of
// bug where a new field is added to a struct but not carried forward during
// base composition or forge resolution.
//
// If this test fails, you added a field to a composable struct without
// updating a merge function. Either:
//  1. Add merge logic for the field in the named function, or
//  2. Add it to composeMergeExemptions with a justification.
func TestComposeMergeCompleteness(t *testing.T) {
	checks := []struct {
		structType reflect.Type
		funcName   string
		sourceFile string
	}{
		{reflect.TypeOf(Harness{}), "mergeBaseIntoChild", "compose.go"},
		{reflect.TypeOf(ForgeConfig{}), "mergeForgeConfigInto", "compose.go"},
		{reflect.TypeOf(ForgeConfig{}), "mergeForgeConfig", "forge.go"},
		{reflect.TypeOf(ValidationLoop{}), "mergeBaseIntoChild", "compose.go"},
		{reflect.TypeOf(ValidationLoop{}), "mergeForgeConfigInto", "compose.go"},
		{reflect.TypeOf(ValidationLoop{}), "mergeForgeConfig", "forge.go"},
	}

	for _, check := range checks {
		structName := check.structType.Name()
		key := structName + "." + check.funcName

		t.Run(key, func(t *testing.T) {
			src := readTestSource(t, check.sourceFile)
			body := extractFuncBody(t, src, check.funcName)
			exemptions := composeMergeExemptions[key]
			if exemptions == nil {
				t.Fatalf("composeMergeExemptions[%q] is missing; add an entry (empty map is fine)", key)
			}

			// Verify no stale exemptions reference non-existent fields.
			for fieldName := range exemptions {
				if !hasStructField(check.structType, fieldName) {
					t.Errorf("composeMergeExemptions[%q] lists %q, "+
						"but %s has no such field (stale exemption?)",
						key, fieldName, structName)
				}
			}

			// Verify every field is referenced or exempted.
			for i := 0; i < check.structType.NumField(); i++ {
				name := check.structType.Field(i).Name

				if reason, ok := exemptions[name]; ok {
					if reason == "" {
						t.Errorf("composeMergeExemptions[%q][%q] "+
							"has an empty justification",
							key, name)
					}
					continue
				}

				if !strings.Contains(body, "."+name) {
					t.Errorf("%s.%s is not referenced in %s "+
						"and not in composeMergeExemptions[%q].\n"+
						"Either update %s to handle this field, "+
						"or add it to composeMergeExemptions with "+
						"a justification.",
						structName, name, check.funcName,
						key, check.funcName)
				}
			}
		})
	}
}

// hasStructField reports whether typ has a field with the given name.
func hasStructField(typ reflect.Type, name string) bool {
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).Name == name {
			return true
		}
	}
	return false
}

// readTestSource reads a Go source file from the same package directory
// as this test file.
func readTestSource(t *testing.T, filename string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	p := filepath.Join(filepath.Dir(thisFile), filename)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	return string(data)
}

// extractFuncBody locates a Go function by name in source text and returns
// its body (from the signature through the closing brace) using brace
// counting. This is intentionally simple — it assumes well-formatted Go
// source without unmatched braces in string literals or comments.
func extractFuncBody(t *testing.T, source, funcName string) string {
	t.Helper()
	lines := strings.Split(source, "\n")
	startLine := -1
	for i, line := range lines {
		if strings.Contains(line, "func ") &&
			strings.Contains(line, funcName+"(") {
			startLine = i
			break
		}
	}
	if startLine == -1 {
		t.Fatalf("function %s not found in source", funcName)
	}

	depth := 0
	started := false
	var body strings.Builder
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				depth++
				started = true
			} else if ch == '}' {
				depth--
			}
		}
		body.WriteString(line)
		body.WriteString("\n")
		if started && depth == 0 {
			break
		}
	}

	return body.String()
}
