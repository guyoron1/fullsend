package runtime

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// --- claudeEventParser tests ---

func TestClaudeEventParser_ToolUse(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/src/main.go"}}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventToolUse {
		t.Errorf("expected EventToolUse, got %d", events[0].Kind)
	}
	if events[0].ToolName != "Read" {
		t.Errorf("expected tool name Read, got %q", events[0].ToolName)
	}
	if events[0].Context != "/src/main.go" {
		t.Errorf("expected context /src/main.go, got %q", events[0].Context)
	}
}

func TestClaudeEventParser_TextEvent(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"text","text":"I will analyze the code now."}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventText {
		t.Errorf("expected EventText, got %d", events[0].Kind)
	}
}

func TestClaudeEventParser_ResultEvent(t *testing.T) {
	line := `{"type":"result","result":"All done"}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventResult {
		t.Errorf("expected EventResult, got %d", events[0].Kind)
	}
}

func TestClaudeEventParser_ErrorEvent(t *testing.T) {
	line := `{"type":"error","error":{"message":"rate limit exceeded"}}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventError {
		t.Errorf("expected EventError, got %d", events[0].Kind)
	}
	if events[0].Text != "rate limit exceeded" {
		t.Errorf("expected error text, got %q", events[0].Text)
	}
}

func TestClaudeEventParser_MixedContent(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"text","text":"Let me read the file."},{"type":"tool_use","name":"Read","input":{"file_path":"/a.go"}}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Kind != EventText {
		t.Errorf("expected EventText first, got %d", events[0].Kind)
	}
	if events[1].Kind != EventToolUse {
		t.Errorf("expected EventToolUse second, got %d", events[1].Kind)
	}
}

func TestClaudeEventParser_UnknownTool(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"SecretTool","input":{"secret":"value"}}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ToolName != "tool" {
		t.Errorf("expected generic 'tool' name, got %q", events[0].ToolName)
	}
	if events[0].Context != "" {
		t.Errorf("expected empty context for unknown tool, got %q", events[0].Context)
	}
}

func TestClaudeEventParser_StreamEventIgnored(t *testing.T) {
	line := `{"type":"stream_event","event":{"type":"content_block_start"}}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 0 {
		t.Errorf("expected 0 events for stream_event, got %d", len(events))
	}
}

func TestClaudeEventParser_MalformedJSON(t *testing.T) {
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(`{not json}`))

	if len(events) != 0 {
		t.Errorf("expected 0 events for malformed JSON, got %d", len(events))
	}
}

func TestClaudeEventParser_EmptyLine(t *testing.T) {
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte{})

	if len(events) != 0 {
		t.Errorf("expected 0 events for empty line, got %d", len(events))
	}
}

func TestClaudeEventParser_UnknownType(t *testing.T) {
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(`{"type":"unknown_type","data":"something"}`))

	if len(events) != 0 {
		t.Errorf("expected 0 events for unknown type, got %d", len(events))
	}
}

func TestClaudeEventParser_BashToolContext(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"make test"}}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Context != "make" {
		t.Errorf("expected context 'make', got %q", events[0].Context)
	}
}

func TestClaudeEventParser_GrepToolContext(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"Grep","input":{"pattern":"func main"}}]}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Context != "func main" {
		t.Errorf("expected context 'func main', got %q", events[0].Context)
	}
}

func TestClaudeEventParser_ErrorEmptyMessage(t *testing.T) {
	line := `{"type":"error","error":{"message":""}}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventError {
		t.Errorf("expected EventError, got %d", events[0].Kind)
	}
	if events[0].Text != "" {
		t.Errorf("expected empty text, got %q", events[0].Text)
	}
}

func TestClaudeEventParser_AssistantMalformedContent(t *testing.T) {
	line := `{"type":"assistant","content":"not an array"}`
	parser := &claudeEventParser{}
	events := parser.ParseLine([]byte(line))

	if len(events) != 0 {
		t.Errorf("expected 0 events for malformed content, got %d", len(events))
	}
}

func TestClaudeEventParser_ImplementsInterface(t *testing.T) {
	var _ EventParser = (*claudeEventParser)(nil)
}

// --- EventRenderer tests ---

func TestEventRenderer_ToolUse(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventToolUse, ToolName: "Read", Context: "/src/main.go"})

	output := buf.String()
	if !strings.Contains(output, "Read: /src/main.go") {
		t.Errorf("expected tool use output, got: %s", output)
	}
	if metrics.ToolCalls.Load() != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCalls.Load())
	}
}

func TestEventRenderer_TextEvent(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventText})

	output := buf.String()
	if !strings.Contains(output, "agent") {
		t.Errorf("expected 'agent' label in text event output, got: %s", output)
	}
	// Text events should not increment tool count.
	if metrics.ToolCalls.Load() != 0 {
		t.Errorf("expected 0 tool calls after text event, got %d", metrics.ToolCalls.Load())
	}
}

func TestEventRenderer_ResultEvent(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventResult})

	output := buf.String()
	if !strings.Contains(output, "result") {
		t.Errorf("expected 'result' label in output, got: %s", output)
	}
}

func TestEventRenderer_ErrorEvent(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventError, Text: "something failed"})

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Errorf("expected 'error' in output, got: %s", output)
	}
	if !strings.Contains(output, "something failed") {
		t.Errorf("expected error text in output, got: %s", output)
	}
}

func TestEventRenderer_ErrorEventEmpty(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventError, Text: ""})

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Errorf("expected 'error' in output, got: %s", output)
	}
}

func TestEventRenderer_ToolCountIncrementsAcrossEvents(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventToolUse, ToolName: "Read", Context: "/a.go"})
	renderer.Render(AgentEvent{Kind: EventText})
	renderer.Render(AgentEvent{Kind: EventToolUse, ToolName: "Bash", Context: "make"})

	if metrics.ToolCalls.Load() != 2 {
		t.Errorf("expected 2 tool calls, got %d", metrics.ToolCalls.Load())
	}

	output := buf.String()
	if !strings.Contains(output, "2 tools") {
		t.Errorf("expected '2 tools' in output, got: %s", output)
	}
}

func TestEventRenderer_ToolUseNoContext(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := &EventRenderer{printer: printer, start: time.Now(), metrics: metrics}

	renderer.Render(AgentEvent{Kind: EventToolUse, ToolName: "tool"})

	output := buf.String()
	if !strings.Contains(output, "tool") {
		t.Errorf("expected 'tool' label, got: %s", output)
	}
	// Should not contain "tool: " (with colon+space) since context is empty.
	if strings.Contains(output, "tool: ") {
		t.Errorf("expected no context separator, got: %s", output)
	}
}

// --- Integration: full stream through progressParser ---

func TestProgressParser_AllEventTypes(t *testing.T) {
	lines := []string{
		`{"type":"assistant","content":[{"type":"text","text":"Let me check the code."}]}`,
		`{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/src/main.go"}}]}`,
		`{"type":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"make test"}}]}`,
		`{"type":"result","result":"All done"}`,
	}

	input := strings.NewReader(strings.Join(lines, "\n"))
	var buf bytes.Buffer
	printer := ui.New(&buf)
	start := time.Now()
	metrics := &RunMetrics{}

	if err := progressParser(input, printer, start, metrics); err != nil {
		t.Fatalf("progressParser returned error: %v", err)
	}

	if metrics.ToolCalls.Load() != 2 {
		t.Errorf("expected 2 tool calls, got %d", metrics.ToolCalls.Load())
	}

	output := buf.String()
	if !strings.Contains(output, "agent") {
		t.Errorf("expected 'agent' for text event, got: %s", output)
	}
	if !strings.Contains(output, "Read: /src/main.go") {
		t.Errorf("expected Read progress, got: %s", output)
	}
	if !strings.Contains(output, "Bash: make") {
		t.Errorf("expected Bash progress, got: %s", output)
	}
	if !strings.Contains(output, "result") {
		t.Errorf("expected result event, got: %s", output)
	}
}

func TestProgressParser_ErrorEventRendered(t *testing.T) {
	lines := []string{
		`{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/a.go"}}]}`,
		`{"type":"error","error":{"message":"rate limit exceeded"}}`,
	}

	input := strings.NewReader(strings.Join(lines, "\n"))
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}

	if err := progressParser(input, printer, time.Now(), metrics); err != nil {
		t.Fatalf("progressParser returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "error") {
		t.Errorf("expected error event in output, got: %s", output)
	}
	if !strings.Contains(output, "rate limit exceeded") {
		t.Errorf("expected error message in output, got: %s", output)
	}
}

func TestProgressParser_MultipleToolsPerMessage(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/a.go"}},{"type":"tool_use","name":"Read","input":{"file_path":"/b.go"}}]}`

	input := strings.NewReader(line)
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}

	if err := progressParser(input, printer, time.Now(), metrics); err != nil {
		t.Fatalf("progressParser returned error: %v", err)
	}

	if metrics.ToolCalls.Load() != 2 {
		t.Errorf("expected 2 tool calls, got %d", metrics.ToolCalls.Load())
	}
}

// Verify the AgentEvent type has expected zero values.
func TestAgentEvent_ZeroValue(t *testing.T) {
	var evt AgentEvent
	if evt.Kind != 0 {
		t.Errorf("expected zero kind, got %d", evt.Kind)
	}
	if evt.ToolName != "" {
		t.Errorf("expected empty tool name, got %q", evt.ToolName)
	}
	if evt.Context != "" {
		t.Errorf("expected empty context, got %q", evt.Context)
	}
	if evt.Text != "" {
		t.Errorf("expected empty text, got %q", evt.Text)
	}
}

// Verify contentItem can unmarshal text field from JSON.
func TestContentItem_TextField(t *testing.T) {
	raw := `{"type":"text","text":"hello world"}`
	var item contentItem
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if item.Type != "text" {
		t.Errorf("expected type 'text', got %q", item.Type)
	}
	if item.Text != "hello world" {
		t.Errorf("expected text 'hello world', got %q", item.Text)
	}
}
