package runtime

// EventKind identifies the type of normalized agent event.
type EventKind int

const (
	// EventToolUse represents an agent tool invocation (Bash, Read, Write, etc.).
	EventToolUse EventKind = iota + 1
	// EventText represents agent-generated text (reasoning, responses).
	EventText
	// EventResult represents the final result of an agent run.
	EventResult
	// EventError represents an error during agent execution.
	EventError
)

// AgentEvent is a runtime-agnostic representation of a single event from an
// agent's execution stream. Any runtime backend (Claude Code, opencode, etc.)
// produces these; the EventRenderer consumes them for display.
type AgentEvent struct {
	Kind     EventKind
	ToolName string // EventToolUse: sanitized tool name (or "tool" for unknown)
	Context  string // EventToolUse: safe display context (file path, binary name, pattern)
	Text     string // EventText/EventResult/EventError: display-safe text
}

// EventParser converts runtime-specific stream data into normalized
// AgentEvents. Each runtime implements this interface to translate its
// native event format (Claude Code stream-json, opencode ndjson, etc.)
// into the common model.
type EventParser interface {
	// ParseLine parses a single line from the runtime's event stream
	// and returns zero or more normalized events. Malformed or
	// unrecognized lines return nil without error.
	ParseLine(line []byte) []AgentEvent
}
