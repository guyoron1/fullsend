package security

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SecretRedactor scans text for API keys, tokens, credentials, and
// sensitive patterns, replacing them with masked versions. Adapted from
// Hermes Agent's agent/redact.py.
type SecretRedactor struct {
	prefixPatterns     []secretPattern
	structuralPatterns []secretPattern
}

type secretPattern struct {
	name  string
	regex *regexp.Regexp
}

// NewSecretRedactor creates a redactor with the default pattern set.
func NewSecretRedactor() *SecretRedactor {
	return &SecretRedactor{
		prefixPatterns:     defaultPrefixPatterns(),
		structuralPatterns: defaultStructuralPatterns(),
	}
}

func (s *SecretRedactor) Name() string { return "secret_redactor" }

func (s *SecretRedactor) Scan(text string) ScanResult {
	if result, ok := s.scanJSON(text); ok {
		return result
	}
	return s.scanText(text)
}

func (s *SecretRedactor) scanText(text string) ScanResult {
	result := ScanResult{Safe: true, Sanitized: text}
	current := text

	// Prefix-based patterns (full match is the secret).
	// Use ReplaceAll to catch duplicate occurrences of the same secret.
	for _, p := range s.prefixPatterns {
		// Deduplicate matches so we report each unique secret once.
		seen := map[string]bool{}
		matches := p.regex.FindAllString(current, -1)
		for _, match := range matches {
			if seen[match] {
				continue
			}
			seen[match] = true
			masked := mask(match)
			result.Findings = append(result.Findings, Finding{
				Scanner:  "secret_redactor",
				Name:     p.name,
				Severity: "critical",
				Detail:   fmt.Sprintf("Redacted %s (%d chars) -> %s", p.name, len(match), masked),
			})
			current = strings.ReplaceAll(current, match, masked)
			result.Safe = false
		}
	}

	// Structural patterns (capture group is the secret)
	for _, p := range s.structuralPatterns {
		// Private key: replace full match (no capture group needed).
		if p.name == "private_key" {
			matches := p.regex.FindAllString(current, -1)
			for _, match := range matches {
				result.Findings = append(result.Findings, Finding{
					Scanner:  "secret_redactor",
					Name:     p.name,
					Severity: "critical",
					Detail:   "Private key block detected and redacted",
				})
				current = strings.ReplaceAll(current, match, "[REDACTED PRIVATE KEY]")
				result.Safe = false
			}
			continue
		}

		// Other structural patterns: process matches in reverse order so
		// byte offsets remain valid as we replace substrings.
		locs := p.regex.FindAllStringSubmatchIndex(current, -1)
		for i := len(locs) - 1; i >= 0; i-- {
			loc := locs[i]
			// Use last non-negative capture group as the secret value.
			// FindAllStringSubmatchIndex returns pairs: [full_start, full_end, group1_start, group1_end, ...]
			nGroups := len(loc)/2 - 1
			if nGroups < 1 {
				continue
			}
			start, end := -1, -1
			for g := nGroups; g >= 1; g-- {
				if loc[g*2] >= 0 && loc[g*2+1] >= 0 {
					start = loc[g*2]
					end = loc[g*2+1]
					break
				}
			}
			if start < 0 || end < 0 {
				continue
			}

			secret := current[start:end]
			masked := mask(secret)
			result.Findings = append(result.Findings, Finding{
				Scanner:  "secret_redactor",
				Name:     p.name,
				Severity: "high",
				Detail:   fmt.Sprintf("Redacted %s (%d chars) -> %s", p.name, len(secret), masked),
			})
			current = current[:start] + masked + current[end:]
			result.Safe = false
		}
	}

	if current != text {
		result.Sanitized = current
	} else {
		result.Sanitized = ""
	}

	return result
}

// scanJSON attempts JSON-aware redaction. If text is valid JSON, it walks
// the parsed tree and applies redaction patterns only to string values,
// preserving structural integrity. Returns (result, true) on success,
// or (ScanResult{}, false) if text is not valid JSON.
func (s *SecretRedactor) scanJSON(text string) (ScanResult, bool) {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return ScanResult{}, false
	}

	var parsed any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return ScanResult{}, false
	}

	aggregate := ScanResult{Safe: true}
	redacted := s.redactValue(parsed, &aggregate)

	if aggregate.Safe {
		return ScanResult{Safe: true}, true
	}

	out, err := json.Marshal(redacted)
	if err != nil {
		// ponytail: shouldn't happen on a value we just parsed; fall back to text scan
		return ScanResult{}, false
	}
	aggregate.Sanitized = string(out)
	return aggregate, true
}

// redactValue recursively walks a parsed JSON value, applying text-level
// redaction to string leaves only.
func (s *SecretRedactor) redactValue(v any, result *ScanResult) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, child := range val {
			// For string values, scan for patterns first
			if str, ok := child.(string); ok {
				r := s.scanText(str)
				if !r.Safe {
					// Pattern matched, use the sanitized value
					result.Safe = false
					result.Findings = append(result.Findings, r.Findings...)
					if r.Sanitized != "" {
						out[k] = r.Sanitized
					} else {
						out[k] = str
					}
				} else if s.isSensitiveKey(k) && len(str) >= 8 {
					// No pattern matched, but key name is sensitive
					masked := mask(str)
					result.Findings = append(result.Findings, Finding{
						Scanner:  "secret_redactor",
						Name:     "json_field",
						Severity: "high",
						Detail:   fmt.Sprintf("Redacted json_field (%d chars) -> %s", len(str), masked),
					})
					out[k] = masked
					result.Safe = false
				} else {
					out[k] = str
				}
			} else {
				out[k] = s.redactValue(child, result)
			}
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, child := range val {
			out[i] = s.redactValue(child, result)
		}
		return out
	case string:
		r := s.scanText(val)
		if !r.Safe {
			result.Safe = false
			result.Findings = append(result.Findings, r.Findings...)
			if r.Sanitized != "" {
				return r.Sanitized
			}
		}
		return val
	default:
		return v
	}
}

// isSensitiveKey checks if a JSON key name suggests a secret value.
func (s *SecretRedactor) isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "key") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "password") ||
		strings.Contains(lower, "credential") ||
		strings.Contains(lower, "auth")
}

func mask(value string) string {
	if len(value) < 10 {
		return "***"
	}
	return value[:4] + "..."
}

func defaultPrefixPatterns() []secretPattern {
	patterns := []struct {
		name    string
		pattern string
	}{
		// Longer prefixes first to match before shorter ones
		{"openai_proj", `sk-proj-[a-zA-Z0-9_-]{20,}`},
		{"anthropic_key", `sk-ant-[a-zA-Z0-9_-]{20,}`},
		{"openai_key", `sk-[a-zA-Z0-9_-]{20,}`},
		{"github_fine_pat", `github_pat_[a-zA-Z0-9_]{22,}`},
		{"github_pat", `ghp_[a-zA-Z0-9_]{36,}`},
		{"github_oauth", `gho_[a-zA-Z0-9_]{36,}`},
		{"github_user_token", `ghu_[a-zA-Z0-9_]{36,}`},
		{"github_server_token", `ghs_[a-zA-Z0-9_.\-]{36,}`},
		{"github_refresh_token", `ghr_[a-zA-Z0-9_]{36,}`},
		{"slack_token", `xox[baprs]-[a-zA-Z0-9-]{10,}`},
		{"google_api_key", `AIza[a-zA-Z0-9_-]{35}`},
		{"aws_access_key", `AKIA[A-Z0-9]{16}`},
		{"stripe_live", `sk_live_[a-zA-Z0-9]{24,}`},
		{"stripe_test", `sk_test_[a-zA-Z0-9]{24,}`},
		{"sendgrid_key", `SG\.[a-zA-Z0-9_-]{22,}\.[a-zA-Z0-9_-]{20,}`},
		{"hf_token", `hf_[a-zA-Z0-9]{34,}`},
		{"npm_token", `npm_[a-zA-Z0-9]{36,}`},
		{"pypi_token", `pypi-[a-zA-Z0-9_-]{50,}`},
		{"gitlab_pat", `glpat-[a-zA-Z0-9_-]{20,}`},
		{"vault_token", `hvs\.[a-zA-Z0-9_-]{24,}`},
		{"age_secret_key", `AGE-SECRET-KEY-[A-Z0-9]{59}`},
	}

	result := make([]secretPattern, len(patterns))
	for i, p := range patterns {
		result[i] = secretPattern{name: p.name, regex: regexp.MustCompile(p.pattern)}
	}
	return result
}

func defaultStructuralPatterns() []secretPattern {
	patterns := []struct {
		name    string
		pattern string
	}{
		{"env_assignment", `(?i)(?:^|\s)(?:export\s+)?((?:[A-Za-z0-9]+_)*(?:KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|PASSWD|AUTH|API_KEY)(?:_[A-Za-z0-9]+)*)\s*=\s*['"]?([^\s'"]{8,})['"]?`},
		{"json_field", `(?:"[^"]*(?i:key|token|secret|password|credential|auth)[^"]*"|'[^']*(?i:key|token|secret|password|credential|auth)[^']*')\s*:\s*(?:"([^"]{8,})"|'([^']{8,})')`},
		{"auth_header", `(?i)(?:Authorization|X-Api-Key|X-Auth-Token)\s*:\s*(?:Bearer\s+)?(\S{8,})`},
		{"private_key", `-----BEGIN\s+(?:RSA\s+|EC\s+|OPENSSH\s+)?PRIVATE KEY-----[\s\S]*?-----END\s+(?:RSA\s+|EC\s+|OPENSSH\s+)?PRIVATE KEY-----`},
		{"db_connection_password", `(?:postgres(?:ql)?|mysql|mongodb|redis)://[^:]+:(.{4,})@[^@\s/]+`},
	}

	result := make([]secretPattern, len(patterns))
	for i, p := range patterns {
		result[i] = secretPattern{name: p.name, regex: regexp.MustCompile("(?m)" + p.pattern)}
	}
	return result
}
