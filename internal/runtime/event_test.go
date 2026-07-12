package runtime

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

func TestClaudeEventParserToolUse(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/src/main.go"}}]}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventToolUse {
		t.Errorf("expected EventToolUse, got %d", events[0].Kind)
	}
	if events[0].Tool != "Read" {
		t.Errorf("expected tool Read, got %q", events[0].Tool)
	}
	if events[0].Context != "/src/main.go" {
		t.Errorf("expected context /src/main.go, got %q", events[0].Context)
	}
}

func TestClaudeEventParserUnknownTool(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"tool_use","name":"EvilTool","input":{"secret":"value"}}]}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Tool != "tool" {
		t.Errorf("expected generic 'tool', got %q", events[0].Tool)
	}
	if events[0].Context != "" {
		t.Errorf("expected empty context for unknown tool, got %q", events[0].Context)
	}
}

func TestClaudeEventParserText(t *testing.T) {
	line := `{"type":"assistant","content":[{"type":"text","text":"Analyzing the code"}]}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventText {
		t.Errorf("expected EventText, got %d", events[0].Kind)
	}
	if events[0].Text != "Analyzing the code" {
		t.Errorf("expected text, got %q", events[0].Text)
	}
}

func TestClaudeEventParserResult(t *testing.T) {
	line := `{"type":"result","result":"All done"}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventResult {
		t.Errorf("expected EventResult, got %d", events[0].Kind)
	}
	if events[0].Text != "All done" {
		t.Errorf("expected result text, got %q", events[0].Text)
	}
}

func TestClaudeEventParserError(t *testing.T) {
	line := `{"type":"error","error":"something went wrong"}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != EventError {
		t.Errorf("expected EventError, got %d", events[0].Kind)
	}
	if events[0].Text != "something went wrong" {
		t.Errorf("expected error text, got %q", events[0].Text)
	}
}

func TestClaudeEventParserMalformedJSON(t *testing.T) {
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(`{not json}`))
	if events != nil {
		t.Errorf("expected nil for malformed JSON, got %v", events)
	}
}

func TestClaudeEventParserStreamEventIgnored(t *testing.T) {
	line := `{"type":"stream_event","event":{"type":"content_block_start"}}`
	parser := ClaudeEventParser{}
	events := parser.Parse([]byte(line))
	if events != nil {
		t.Errorf("expected nil for stream_event, got %v", events)
	}
}

func TestClaudeEventParserMultipleToolUse(t *testing.T) {
	items := []map[string]interface{}{
		{"type": "tool_use", "name": "Read", "input": map[string]string{"file_path": "/a.go"}},
		{"type": "tool_use", "name": "Bash", "input": map[string]string{"command": "make test"}},
	}
	content, _ := json.Marshal(items)
	line, _ := json.Marshal(map[string]interface{}{
		"type":    "assistant",
		"content": json.RawMessage(content),
	})

	parser := ClaudeEventParser{}
	events := parser.Parse(line)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Tool != "Read" || events[1].Tool != "Bash" {
		t.Errorf("unexpected tools: %q, %q", events[0].Tool, events[1].Tool)
	}
}

func TestEventRendererToolUse(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := NewEventRenderer(printer, time.Now(), metrics)

	renderer.Render(AgentEvent{Kind: EventToolUse, Tool: "Read", Context: "/src/main.go"})

	if metrics.ToolCalls.Load() != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCalls.Load())
	}
	output := buf.String()
	if !strings.Contains(output, "Read: /src/main.go") {
		t.Errorf("expected Read progress, got: %s", output)
	}
}

func TestEventRendererText(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := NewEventRenderer(printer, time.Now(), metrics)

	renderer.Render(AgentEvent{Kind: EventText, Text: "thinking"})

	output := buf.String()
	if !strings.Contains(output, "thinking") {
		t.Errorf("expected text output, got: %s", output)
	}
}

func TestEventRendererError(t *testing.T) {
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	renderer := NewEventRenderer(printer, time.Now(), metrics)

	renderer.Render(AgentEvent{Kind: EventError, Text: "bad thing"})

	output := buf.String()
	if !strings.Contains(output, "bad thing") {
		t.Errorf("expected error output, got: %s", output)
	}
}

func TestParseAndRenderEndToEnd(t *testing.T) {
	lines := []string{
		`{"type":"assistant","content":[{"type":"tool_use","name":"Read","input":{"file_path":"/src/main.go"}}]}`,
		`{"type":"assistant","content":[{"type":"text","text":"Analyzing code"}]}`,
		`{"type":"assistant","content":[{"type":"tool_use","name":"Bash","input":{"command":"make test"}}]}`,
		`{"type":"result","result":"All done"}`,
		`{"type":"error","error":"oops"}`,
	}

	input := strings.NewReader(strings.Join(lines, "\n"))
	var buf bytes.Buffer
	printer := ui.New(&buf)
	metrics := &RunMetrics{}
	parser := ClaudeEventParser{}
	renderer := NewEventRenderer(printer, time.Now(), metrics)

	if err := parseAndRender(input, parser, renderer); err != nil {
		t.Fatalf("parseAndRender returned error: %v", err)
	}

	if metrics.ToolCalls.Load() != 2 {
		t.Errorf("expected 2 tool calls, got %d", metrics.ToolCalls.Load())
	}

	output := buf.String()
	if !strings.Contains(output, "Read: /src/main.go") {
		t.Errorf("expected Read progress, got: %s", output)
	}
	if !strings.Contains(output, "Bash: make") {
		t.Errorf("expected Bash progress, got: %s", output)
	}
	if !strings.Contains(output, "Analyzing code") {
		t.Errorf("expected text event, got: %s", output)
	}
	if !strings.Contains(output, "All done") {
		t.Errorf("expected result event, got: %s", output)
	}
	if !strings.Contains(output, "oops") {
		t.Errorf("expected error event, got: %s", output)
	}
}
