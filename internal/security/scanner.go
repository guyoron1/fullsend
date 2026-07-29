// Package security provides input sanitization and output scanning for
// fullsend's agent entrypoints. These scanners run at the boundaries:
//
//   - Input boundary: before untrusted text (issue bodies, PR descriptions,
//     code comments, context files) reaches agent processing
//   - Output boundary: before agent-generated text (PR comments, issue
//     comments) is posted via the forge API
//
// The scanners are adapted from Hermes Agent's production security controls,
// ported to Go for integration into fullsend's CLI entrypoints.
//
// See: experiments/hermes-security-patterns/ for the Python prototypes
// and evaluation results.
package security

import (
	"encoding/json"
	"fmt"
)

// Finding represents a single security issue detected by a scanner.
type Finding struct {
	Scanner  string // "secret_redactor", "ssrf_validator", "context_injection", "unicode_normalizer"
	Name     string // pattern name or category
	Severity string // "critical", "high", "medium"
	Detail   string // human-readable description
	Position int    // byte offset in original text, -1 if N/A
}

// ScanResult holds the outcome of a security scan.
type ScanResult struct {
	Safe      bool
	Findings  []Finding
	Sanitized string // cleaned/redacted version of input (empty if unchanged)
}

// Scanner is the interface for all security scanners.
type Scanner interface {
	// Name returns the scanner identifier.
	Name() string

	// Scan checks text for security issues. Returns a ScanResult with
	// findings and optionally a sanitized version of the input.
	Scan(text string) ScanResult
}

// Pipeline chains multiple scanners in sequence. Each scanner's sanitized
// output feeds into the next scanner's input.
type Pipeline struct {
	scanners []Scanner
}

// NewPipeline creates a scanner pipeline from the given scanners.
// Scanners run in order; place normalizers first, detectors after.
func NewPipeline(scanners ...Scanner) *Pipeline {
	return &Pipeline{scanners: scanners}
}

// Scan runs all scanners in sequence. Returns the aggregate result.
// The pipeline is fail-open for sanitization (each scanner transforms
// the text) but fail-closed for safety (any scanner marking unsafe
// makes the whole result unsafe).
func (p *Pipeline) Scan(text string) ScanResult {
	aggregate := ScanResult{Safe: true, Sanitized: text}
	current := text

	for _, s := range p.scanners {
		result := s.Scan(current)

		aggregate.Findings = append(aggregate.Findings, result.Findings...)
		if !result.Safe {
			aggregate.Safe = false
		}
		if result.Sanitized != "" {
			current = result.Sanitized
			aggregate.Sanitized = current
		}
	}

	if aggregate.Sanitized == text {
		aggregate.Sanitized = "" // no changes
	}

	return aggregate
}

// InputPipeline returns the standard input scanning pipeline for
// untrusted text entering the agent. Order matters:
//  1. UnicodeNormalizer — strip invisible chars, normalize fullwidth
//  2. ContextInjectionScanner — detect prompt injection patterns
func InputPipeline() *Pipeline {
	return NewPipeline(
		NewUnicodeNormalizer(),
		NewContextInjectionScanner(),
	)
}

// OutputPipeline returns the standard output scanning pipeline for
// agent-generated text before posting to the forge.
//  1. UnicodeNormalizer — strip invisible chars, normalize fullwidth
//  2. SecretRedactor — redact API keys, tokens, credentials
func OutputPipeline() *Pipeline {
	return NewPipeline(
		NewUnicodeNormalizer(),
		NewSecretRedactor(),
	)
}

// ScanJSON performs JSON-structure-aware scanning. It parses the input as
// JSON, recursively walks the tree, and applies scanners only to string
// leaf values. This preserves JSON structural integrity — redactions
// inside string values cannot break the enclosing JSON syntax.
//
// If the input is not valid JSON, ScanJSON falls back to text-based
// Scan(). If the input is valid JSON but the sanitized output would
// somehow be invalid (should not happen with the tree-walk approach,
// but checked defensively), it falls back to the original content and
// reports a warning finding.
func (p *Pipeline) ScanJSON(data []byte) (ScanResult, []byte) {
	// Try to parse as JSON. If it fails, fall back to text-based scan.
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		result := p.Scan(string(data))
		out := string(data)
		if result.Sanitized != "" {
			out = result.Sanitized
		}
		return result, []byte(out)
	}

	aggregate := ScanResult{Safe: true}

	// Walk the parsed JSON tree and apply scanners to string leaves.
	sanitized := p.walkJSON(parsed, &aggregate)

	if aggregate.Safe {
		// No findings — return original data unchanged.
		return aggregate, data
	}

	// Marshal the sanitized tree back to JSON, preserving structure.
	out, err := json.Marshal(sanitized)
	if err != nil {
		// Defensive fallback: marshalling failed (should not happen).
		aggregate.Findings = append(aggregate.Findings, Finding{
			Scanner:  "json_integrity",
			Name:     "marshal_error",
			Severity: "high",
			Detail:   fmt.Sprintf("JSON re-marshal failed after redaction: %v; falling back to original", err),
		})
		return aggregate, data
	}

	// Defensive: validate the output is still valid JSON.
	var check any
	if err := json.Unmarshal(out, &check); err != nil {
		// Should never happen, but if it does, fall back to original.
		aggregate.Findings = append(aggregate.Findings, Finding{
			Scanner:  "json_integrity",
			Name:     "validation_error",
			Severity: "high",
			Detail:   fmt.Sprintf("Sanitized JSON failed re-parse: %v; falling back to original", err),
		})
		return aggregate, data
	}

	aggregate.Sanitized = string(out)
	return aggregate, out
}

// walkJSON recursively walks a parsed JSON value and applies the pipeline's
// scanners to every string leaf. Non-string leaves are returned unchanged.
func (p *Pipeline) walkJSON(v any, aggregate *ScanResult) any {
	switch val := v.(type) {
	case string:
		result := p.Scan(val)
		aggregate.Findings = append(aggregate.Findings, result.Findings...)
		if !result.Safe {
			aggregate.Safe = false
		}
		if result.Sanitized != "" {
			return result.Sanitized
		}
		return val

	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			out[k] = p.walkJSON(child, aggregate)
		}
		return out

	case []any:
		out := make([]any, len(val))
		for i, child := range val {
			out[i] = p.walkJSON(child, aggregate)
		}
		return out

	default:
		// Numbers, booleans, nil — return unchanged.
		return val
	}
}

// HasCriticalFindings reports whether any finding has critical severity.
func HasCriticalFindings(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == "critical" {
			return true
		}
	}
	return false
}
