package runtime

import (
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

const (
	// maxErrorDisplay is the maximum length of error text emitted in
	// progress output and GHA annotations. Matches the
	// maxTranscriptErrorLength limit used for transcript error handling.
	maxErrorDisplay = 2000
)

// EventRenderer formats normalized AgentEvents for display via a Printer.
// It is runtime-agnostic — any EventParser's output can be rendered
// consistently through this type.
type EventRenderer struct {
	printer *ui.Printer
	start   time.Time
	metrics *RunMetrics
	isCI    bool
}

// NewEventRenderer creates a renderer that writes formatted events to the
// given printer. The start time and metrics are used for elapsed-time and
// tool-count annotations on each line.
func NewEventRenderer(printer *ui.Printer, start time.Time, metrics *RunMetrics) *EventRenderer {
	return &EventRenderer{
		printer: printer,
		start:   start,
		metrics: metrics,
		isCI:    os.Getenv("GITHUB_ACTIONS") == "true",
	}
}

// Render formats and emits a single AgentEvent.
func (r *EventRenderer) Render(evt AgentEvent) {
	switch evt.Kind {
	case EventToolUse:
		count := r.metrics.ToolCalls.Add(1)
		emitToolProgress(r.printer, evt.ToolName, evt.Context, r.start, count, r.isCI)
	case EventText:
		r.emitEvent("agent")
	case EventResult:
		r.emitEvent("result")
	case EventError:
		text := truncateErrorDisplay(sanitizeOutput(evt.Text))
		if text != "" {
			r.emitEvent("error: " + text)
		} else {
			r.emitEvent("error")
		}
	default:
		// Unknown or zero-value EventKind — silently drop. New event
		// kinds can be added to the switch as the model evolves.
	}
}

// truncateErrorDisplay trims error text to maxErrorDisplay. If truncated,
// walks back to a valid UTF-8 rune boundary before appending an indicator.
func truncateErrorDisplay(msg string) string {
	if len(msg) <= maxErrorDisplay {
		return msg
	}
	truncated := msg[:maxErrorDisplay]
	for len(truncated) > 0 && !utf8.Valid([]byte(truncated)) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated + "… (truncated)"
}

// emitEvent formats and prints a non-tool event line with elapsed time.
func (r *EventRenderer) emitEvent(label string) {
	elapsed := time.Since(r.start).Truncate(time.Second)
	toolCount := r.metrics.ToolCalls.Load()
	msg := fmt.Sprintf("%s (%s, %d tools)", label, elapsed, toolCount)
	msg = sanitizeOutput(msg)
	if r.isCI {
		fmt.Fprintf(os.Stderr, "::notice::%s\n", msg)
	}
	r.printer.Heartbeat(msg)
}
