package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// AgentInfo represents a discovered agent with role and slug identifiers.
type AgentInfo struct {
	Role     string `yaml:"role"`
	Slug     string `yaml:"slug"`
	Filename string `yaml:"-"` // source filename, not persisted
}

// HarnessWrapperFile represents a harness wrapper file's content.
type HarnessWrapperFile struct {
	Role string `yaml:"role"`
	Slug string `yaml:"slug"`
}

// ConfigYAML represents the legacy config.yaml structure.
type ConfigYAML struct {
	Agents []string `yaml:"agents"`
}

// MockForgeClient simulates forge client interactions for testing agent slug
// discovery without requiring real repository access.
type MockForgeClient struct {
	harnessFiles      map[string][]byte // filename → raw YAML content
	harnessDir        bool              // whether harness directory exists
	harnessError      error             // hard error on harness directory listing
	configYAML        []byte            // raw config.yaml content
	configAccessed    bool              // tracks whether config.yaml was read
	fileReadErrors    map[string]error   // per-file read errors
}

// MockForgeOption configures a MockForgeClient.
type MockForgeOption func(*MockForgeClient)

// NewMockForgeClient creates a MockForgeClient with the given options.
func NewMockForgeClient(opts ...MockForgeOption) *MockForgeClient {
	m := &MockForgeClient{
		harnessFiles:   make(map[string][]byte),
		fileReadErrors: make(map[string]error),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// withHarnessFiles configures the mock with harness wrapper files.
func withHarnessFiles(files map[string]HarnessWrapperFile) MockForgeOption {
	return func(m *MockForgeClient) {
		m.harnessDir = true
		for name, f := range files {
			data, _ := yaml.Marshal(f)
			m.harnessFiles[name] = data
		}
	}
}

// withoutHarnessDir configures the mock with no harness directory.
func withoutHarnessDir() MockForgeOption {
	return func(m *MockForgeClient) {
		m.harnessDir = false
	}
}

// withConfigAgents configures the mock with a config.yaml containing an agents block.
func withConfigAgents(agents []string) MockForgeOption {
	return func(m *MockForgeClient) {
		cfg := ConfigYAML{Agents: agents}
		data, _ := yaml.Marshal(cfg)
		m.configYAML = data
	}
}

// withEmptyConfig configures the mock with an empty config.yaml (no agents block).
func withEmptyConfig() MockForgeOption {
	return func(m *MockForgeClient) {
		m.configYAML = []byte("{}")
	}
}

// withMalformedConfig configures the mock with malformed YAML config.
func withMalformedConfig() MockForgeOption {
	return func(m *MockForgeClient) {
		m.configYAML = []byte("agents: [invalid yaml: {{broken")
	}
}

// withHarnessError configures the mock to return a hard error on harness dir listing.
func withHarnessError(err error) MockForgeOption {
	return func(m *MockForgeClient) {
		m.harnessDir = true
		m.harnessError = err
	}
}

// withFileReadErrors configures per-file read errors for partial failure testing.
func withFileReadErrors(errors map[string]error) MockForgeOption {
	return func(m *MockForgeClient) {
		m.fileReadErrors = errors
	}
}

// ConfigYAMLAccessed returns whether config.yaml was read during discovery.
func (m *MockForgeClient) ConfigYAMLAccessed() bool {
	return m.configAccessed
}

// ListHarnessDir lists files in the harness directory.
func (m *MockForgeClient) ListHarnessDir() ([]string, error) {
	if m.harnessError != nil {
		return nil, m.harnessError
	}
	if !m.harnessDir {
		return nil, fmt.Errorf("harness directory not found")
	}
	var names []string
	for name := range m.harnessFiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// ReadHarnessFile reads a single harness wrapper file.
func (m *MockForgeClient) ReadHarnessFile(name string) ([]byte, error) {
	if err, ok := m.fileReadErrors[name]; ok {
		return nil, err
	}
	data, ok := m.harnessFiles[name]
	if !ok {
		return nil, fmt.Errorf("file not found: %s", name)
	}
	return data, nil
}

// ReadConfigYAML reads the legacy config.yaml content.
func (m *MockForgeClient) ReadConfigYAML() ([]byte, error) {
	m.configAccessed = true
	if m.configYAML == nil {
		return nil, fmt.Errorf("config.yaml not found")
	}
	return m.configYAML, nil
}

// Printer captures output for test verification.
type Printer struct {
	buf *bytes.Buffer
}

// NewPrinter creates a Printer backed by the given buffer.
func NewPrinter(buf *bytes.Buffer) *Printer {
	return &Printer{buf: buf}
}

// Printf writes formatted output to the printer buffer.
func (p *Printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(p.buf, format, args...)
}

// Writer returns the underlying io.Writer.
func (p *Printer) Writer() io.Writer {
	return p.buf
}

// DiscoverAgentSlugs discovers agent slugs using the harness-first model.
// It first attempts to read agents from harness wrapper files. If that fails
// or yields no valid agents, it falls back to the legacy config.yaml agents block.
func DiscoverAgentSlugs(ctx context.Context, forge *MockForgeClient, configRepo, ref string, printer *Printer) ([]AgentInfo, error) {
	_ = ctx // context used for cancellation in production

	// Step 1: Try harness discovery
	harnessAgents, harnessErr := discoverFromHarness(forge, printer)

	if harnessErr == nil && len(harnessAgents) > 0 {
		// Harness discovery succeeded — return without consulting config.yaml
		return deduplicateAgents(harnessAgents, printer), nil
	}

	// Log warning if harness discovery encountered errors
	if harnessErr != nil {
		printer.Printf("warning: harness discovery failed: %v, falling back to config.yaml\n", harnessErr)
	}

	// Step 2: Fall back to legacy config.yaml
	agents, err := discoverFromConfigYAML(forge, printer)
	if err != nil {
		// Config.yaml also failed — return nil without error
		return nil, nil
	}

	return agents, nil
}

// discoverFromHarness reads agent info from harness wrapper files.
func discoverFromHarness(forge *MockForgeClient, printer *Printer) ([]AgentInfo, error) {
	fileNames, err := forge.ListHarnessDir()
	if err != nil {
		return nil, err
	}

	var agents []AgentInfo
	for _, name := range fileNames {
		data, readErr := forge.ReadHarnessFile(name)
		if readErr != nil {
			// Partial error — skip this file, continue with others
			printer.Printf("warning: failed to read harness file %s: %v\n", name, readErr)
			continue
		}

		var wrapper HarnessWrapperFile
		if parseErr := yaml.Unmarshal(data, &wrapper); parseErr != nil {
			printer.Printf("warning: failed to parse harness file %s: %v\n", name, parseErr)
			continue
		}

		// Both empty → silent skip (placeholder/template file)
		if wrapper.Role == "" && wrapper.Slug == "" {
			continue
		}

		// Role present but no slug → skip with warning
		if wrapper.Role != "" && wrapper.Slug == "" {
			printer.Printf("warning: harness file %s has role %q but no slug, skipping\n", name, wrapper.Role)
			continue
		}

		// Slug present but no role → skip with warning
		if wrapper.Role == "" && wrapper.Slug != "" {
			printer.Printf("warning: harness file %s has slug %q but no role, skipping\n", name, wrapper.Slug)
			continue
		}

		agents = append(agents, AgentInfo{
			Role:     wrapper.Role,
			Slug:     wrapper.Slug,
			Filename: name,
		})
	}

	return agents, nil
}

// discoverFromConfigYAML reads agent info from the legacy config.yaml agents block.
func discoverFromConfigYAML(forge *MockForgeClient, printer *Printer) ([]AgentInfo, error) {
	data, err := forge.ReadConfigYAML()
	if err != nil {
		return nil, err
	}

	var cfg ConfigYAML
	if parseErr := yaml.Unmarshal(data, &cfg); parseErr != nil {
		return nil, parseErr
	}

	if len(cfg.Agents) == 0 {
		return nil, nil
	}

	printer.Printf("warning: using deprecated config.yaml agents block, migrate to harness wrapper files\n")

	var agents []AgentInfo
	for _, slug := range cfg.Agents {
		agents = append(agents, AgentInfo{
			Role: slug,
			Slug: slug,
		})
	}

	return agents, nil
}

// deduplicateAgents removes duplicate roles, keeping the first occurrence
// sorted by Role then Filename.
func deduplicateAgents(agents []AgentInfo, printer *Printer) []AgentInfo {
	// Sort by Role, then Filename for deterministic ordering
	sort.Slice(agents, func(i, j int) bool {
		if agents[i].Role != agents[j].Role {
			return agents[i].Role < agents[j].Role
		}
		return agents[i].Filename < agents[j].Filename
	})

	seen := make(map[string]bool)
	var deduped []AgentInfo
	for _, a := range agents {
		if seen[a.Role] {
			printer.Printf("info: duplicate role %q from file %s, already seen — skipping\n", a.Role, a.Filename)
			continue
		}
		seen[a.Role] = true
		deduped = append(deduped, a)
	}

	return deduped
}

// FilterAgentsByAppSet filters agents by app-set membership.
// In production, this would check app-set configuration; here it uses
// a simple name-contains heuristic for test demonstration.
func FilterAgentsByAppSet(agents []AgentInfo, appSet string) []AgentInfo {
	var filtered []AgentInfo
	for _, a := range agents {
		if strings.Contains(a.Role, appSet) || strings.Contains(a.Slug, appSet) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// InstallSetup simulates the install setup function that uses agent slug discovery
// to initiate application configuration.
func InstallSetup(ctx context.Context, forge *MockForgeClient, configRepo, ref string, printer *Printer) ([]AgentInfo, error) {
	agents, err := DiscoverAgentSlugs(ctx, forge, configRepo, ref, printer)
	if err != nil {
		return nil, fmt.Errorf("install setup: agent discovery failed: %w", err)
	}
	return agents, nil
}
