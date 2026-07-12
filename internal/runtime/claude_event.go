package runtime

import "encoding/json"

// ClaudeEventParser implements EventParser for Claude Code's stream-json format.
type ClaudeEventParser struct{}

func (ClaudeEventParser) Parse(line []byte) []AgentEvent {
	var evt streamEvent
	if err := json.Unmarshal(line, &evt); err != nil {
		return nil
	}

	switch evt.Type {
	case "assistant":
		return parseClaudeAssistant(line)
	case "result":
		return parseClaudeResult(line)
	case "error":
		return parseClaudeError(line)
	}
	return nil
}

// textContentItem extracts the text field from a content block of type "text".
type textContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func parseClaudeAssistant(line []byte) []AgentEvent {
	var msg assistantMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(msg.Content, &rawItems); err != nil {
		return nil
	}

	var events []AgentEvent
	for _, raw := range rawItems {
		var base struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &base) != nil {
			continue
		}

		switch base.Type {
		case "tool_use":
			var item contentItem
			if json.Unmarshal(raw, &item) != nil {
				continue
			}
			toolName := item.Name
			var ctx string
			if !allowedTools[toolName] {
				toolName = "tool"
			} else {
				ctx = extractSafeContext(item.Name, item.Input)
			}
			events = append(events, AgentEvent{
				Kind:    EventToolUse,
				Tool:    toolName,
				Context: ctx,
			})
		case "text":
			var item textContentItem
			if json.Unmarshal(raw, &item) != nil || item.Text == "" {
				continue
			}
			events = append(events, AgentEvent{
				Kind: EventText,
				Text: item.Text,
			})
		}
	}
	return events
}

// resultMessage extracts the result field from a top-level result event.
type resultMessage struct {
	Result string `json:"result"`
}

// errorMessage extracts the error field from a top-level error event.
type errorMessage struct {
	Error string `json:"error"`
}

func parseClaudeResult(line []byte) []AgentEvent {
	var msg resultMessage
	if err := json.Unmarshal(line, &msg); err != nil || msg.Result == "" {
		return nil
	}
	return []AgentEvent{{Kind: EventResult, Text: msg.Result}}
}

func parseClaudeError(line []byte) []AgentEvent {
	var msg errorMessage
	if err := json.Unmarshal(line, &msg); err != nil || msg.Error == "" {
		return nil
	}
	return []AgentEvent{{Kind: EventError, Text: msg.Error}}
}
