package runtime

import (
	"fmt"
	"os"
	"time"

	"github.com/fullsend-ai/fullsend/internal/ui"
)

// EventRenderer formats AgentEvents and emits them via a ui.Printer.
type EventRenderer struct {
	printer *ui.Printer
	start   time.Time
	metrics *RunMetrics
	isCI    bool
}

// NewEventRenderer creates a renderer bound to the given printer and clock.
func NewEventRenderer(printer *ui.Printer, start time.Time, metrics *RunMetrics) *EventRenderer {
	return &EventRenderer{
		printer: printer,
		start:   start,
		metrics: metrics,
		isCI:    os.Getenv("GITHUB_ACTIONS") == "true",
	}
}

// Render emits a single AgentEvent through the printer.
func (r *EventRenderer) Render(evt AgentEvent) {
	switch evt.Kind {
	case EventToolUse:
		count := r.metrics.ToolCalls.Add(1)
		emitToolProgress(r.printer, evt.Tool, evt.Context, r.start, count, r.isCI)
	case EventText, EventResult:
		elapsed := time.Since(r.start).Truncate(time.Second)
		msg := sanitizeOutput(fmt.Sprintf("%s (%s)", evt.Text, elapsed))
		if r.isCI {
			fmt.Fprintf(os.Stderr, "::notice::%s\n", msg)
		}
		r.printer.Heartbeat(msg)
	case EventError:
		r.printer.StepWarn(sanitizeOutput(evt.Text))
	}
}
