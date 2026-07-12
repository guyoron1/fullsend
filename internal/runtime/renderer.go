package runtime

import (
	"fmt"
	"os"
	"time"

	"github.com/fullsend-ai/fullsend/internal/ui"
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
		text := sanitizeOutput(evt.Text)
		if text != "" {
			r.emitEvent("error: " + text)
		} else {
			r.emitEvent("error")
		}
	}
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
