package harness

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// extractFuncBody finds the named function in source and returns its full text
// (from "func name(" through the matching closing brace). Returns "" if not found.
func extractFuncBody(source, funcName string) string {
	marker := "func " + funcName + "("
	idx := strings.Index(source, marker)
	if idx < 0 {
		return ""
	}
	rest := source[idx:]
	braceIdx := strings.Index(rest, "{")
	if braceIdx < 0 {
		return ""
	}
	depth := 1
	pos := braceIdx + 1
	for pos < len(rest) && depth > 0 {
		switch rest[pos] {
		case '{':
			depth++
		case '}':
			depth--
		}
		pos++
	}
	return rest[:pos]
}

func TestComposableFieldsCoveredByMergeFunctions(t *testing.T) {
	// Exemption maps: fields deliberately excluded from merging.
	// To exempt a new field, add it here with a comment explaining why.
	harnessExemptions := map[string]string{
		"Base":                   "consumed by LoadWithBase before merging",
		"AllowedRemoteResources": "security: child must declare its own allowlist",
		"AllowRuntimeFetch":      "security: not merged from base to prevent privilege escalation",
		"MaxRuntimeFetches":      "security: not merged from base to prevent privilege escalation",
	}
	forgeConfigIntoExemptions := map[string]string{}
	forgeConfigExemptions := map[string]string{}

	composeSource, err := os.ReadFile("compose.go")
	if err != nil {
		t.Fatalf("reading compose.go: %v", err)
	}
	forgeSource, err := os.ReadFile("forge.go")
	if err != nil {
		t.Fatalf("reading forge.go: %v", err)
	}

	// Extract merge function bodies
	mergeBaseBody := extractFuncBody(string(composeSource), "mergeBaseIntoChild")
	if mergeBaseBody == "" {
		t.Fatal("could not find mergeBaseIntoChild in compose.go")
	}
	mergeForgeIntoBody := extractFuncBody(string(composeSource), "mergeForgeConfigInto")
	if mergeForgeIntoBody == "" {
		t.Fatal("could not find mergeForgeConfigInto in compose.go")
	}
	mergeForgeBody := extractFuncBody(string(forgeSource), "mergeForgeConfig")
	if mergeForgeBody == "" {
		t.Fatal("could not find mergeForgeConfig in forge.go")
	}

	// Check Harness fields in mergeBaseIntoChild
	harnessType := reflect.TypeOf(Harness{})
	for i := 0; i < harnessType.NumField(); i++ {
		field := harnessType.Field(i)
		if _, exempt := harnessExemptions[field.Name]; exempt {
			continue
		}
		if !strings.Contains(mergeBaseBody, field.Name) {
			t.Errorf("Harness field %q not found in mergeBaseIntoChild and not exempted.\n"+
				"Add it to mergeBaseIntoChild in compose.go, or add it to the exemption\n"+
				"map in this test with a comment explaining why carry-forward is not needed.",
				field.Name)
		}
	}

	// Check ForgeConfig fields in mergeForgeConfigInto
	forgeConfigType := reflect.TypeOf(ForgeConfig{})
	for i := 0; i < forgeConfigType.NumField(); i++ {
		field := forgeConfigType.Field(i)
		if _, exempt := forgeConfigIntoExemptions[field.Name]; exempt {
			continue
		}
		if !strings.Contains(mergeForgeIntoBody, field.Name) {
			t.Errorf("ForgeConfig field %q not found in mergeForgeConfigInto and not exempted.\n"+
				"Add it to mergeForgeConfigInto in compose.go, or add it to the exemption\n"+
				"map in this test with a comment explaining why carry-forward is not needed.",
				field.Name)
		}
	}

	// Check ForgeConfig fields in mergeForgeConfig
	for i := 0; i < forgeConfigType.NumField(); i++ {
		field := forgeConfigType.Field(i)
		if _, exempt := forgeConfigExemptions[field.Name]; exempt {
			continue
		}
		if !strings.Contains(mergeForgeBody, field.Name) {
			t.Errorf("ForgeConfig field %q not found in mergeForgeConfig and not exempted.\n"+
				"Add it to mergeForgeConfig in forge.go, or add it to the exemption\n"+
				"map in this test with a comment explaining why carry-forward is not needed.",
				field.Name)
		}
	}
}
