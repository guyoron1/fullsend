package runtime

// EventKind classifies the type of agent event in the normalized stream.
type EventKind int

const (
	EventToolUse EventKind = iota
	EventText
	EventResult
	EventError
)

// AgentEvent is a runtime-agnostic representation of a single event from an
// agent's execution stream. Runtimes parse their native format into this
// model; the renderer consumes it without knowledge of the source format.
type AgentEvent struct {
	Kind    EventKind
	Tool    string // tool name for EventToolUse
	Context string // safe display context (file path, binary name, pattern)
	Text    string // message body for EventText/EventResult/EventError
}

// EventParser converts a raw NDJSON line from a runtime's event stream into
// zero or more AgentEvents. Implementations are runtime-specific.
type EventParser interface {
	// Parse returns events extracted from a single line of stream output.
	// Returns nil if the line is not relevant or cannot be parsed.
	Parse(line []byte) []AgentEvent
}
