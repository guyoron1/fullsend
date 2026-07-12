package runtime

import "encoding/json"

// claudeEventParser implements EventParser for Claude Code's stream-json format.
// It translates NDJSON events from --output-format stream-json into normalized
// AgentEvents.
type claudeEventParser struct{}

// resultEvent represents a Claude Code stream-json result line.
type resultEvent struct {
	Type   string `json:"type"`
	Result string `json:"result"`
}

// errorEventWrapper represents a Claude Code stream-json error line.
type errorEventWrapper struct {
	Type  string     `json:"type"`
	Error errorInner `json:"error"`
}

type errorInner struct {
	Message string `json:"message"`
}

// ParseLine parses a single NDJSON line from Claude Code's stream-json output
// and returns zero or more normalized AgentEvents.
func (p *claudeEventParser) ParseLine(line []byte) []AgentEvent {
	if len(line) == 0 {
		return nil
	}

	var evt streamEvent
	if err := json.Unmarshal(line, &evt); err != nil {
		return nil
	}

	switch evt.Type {
	case "assistant":
		return p.parseAssistant(line)
	case "result":
		return p.parseResult(line)
	case "error":
		return p.parseError(line)
	default:
		return nil
	}
}

// parseAssistant extracts tool_use and text content items from assistant messages.
func (p *claudeEventParser) parseAssistant(line []byte) []AgentEvent {
	var msg assistantMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil
	}

	var items []contentItem
	if err := json.Unmarshal(msg.Content, &items); err != nil {
		return nil
	}

	var events []AgentEvent
	for _, item := range items {
		switch item.Type {
		case "tool_use":
			toolName := item.Name
			var ctx string
			if !allowedTools[toolName] {
				toolName = "tool"
			} else {
				ctx = extractSafeContext(item.Name, item.Input)
			}
			events = append(events, AgentEvent{
				Kind:     EventToolUse,
				ToolName: toolName,
				Context:  ctx,
			})
		case "text":
			events = append(events, AgentEvent{
				Kind: EventText,
			})
		}
	}
	return events
}

// parseResult extracts the completion result.
func (p *claudeEventParser) parseResult(line []byte) []AgentEvent {
	var r resultEvent
	if err := json.Unmarshal(line, &r); err != nil {
		return nil
	}
	return []AgentEvent{{Kind: EventResult}}
}

// parseError extracts error information.
func (p *claudeEventParser) parseError(line []byte) []AgentEvent {
	var e errorEventWrapper
	if err := json.Unmarshal(line, &e); err != nil {
		return nil
	}
	return []AgentEvent{{Kind: EventError, Text: e.Error.Message}}
}

// Ensure claudeEventParser implements EventParser.
var _ EventParser = (*claudeEventParser)(nil)
