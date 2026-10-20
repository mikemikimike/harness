package chat

import (
	"encoding/json"
	"strings"

	"github.com/stubbedev/harness/internal/agent"
	"github.com/stubbedev/harness/internal/ui/styles"
)

// AgentControlToolRenderContext renders the dispatcher tool acting on
// background agents already running: a message sent to them or a cancel.
// The header names the action and its targets; the body is the delivery
// acknowledgement.
type AgentControlToolRenderContext struct {
	kind agent.AgentCallKind
}

// label names the call by its action, in the progressive while it runs.
func (a *AgentControlToolRenderContext) label(running bool) string {
	switch {
	case a.kind == agent.AgentCallCancel && running:
		return "Cancelling agents"
	case a.kind == agent.AgentCallCancel:
		return "Cancelled agents"
	case running:
		return "Messaging agents"
	default:
		return "Messaged agents"
	}
}

// RenderTool implements the [ToolRenderer] interface.
func (a *AgentControlToolRenderContext) RenderTool(sty *styles.Styles, width int, opts *ToolRenderOpts) string {
	return renderStandardTool(sty, width, opts, opts.Name, func() ([]string, bool) {
		var params agent.AgentDispatchParams
		if err := json.Unmarshal([]byte(opts.ToolCall.Input), &params); err != nil {
			return nil, false
		}
		targets := "all running"
		if len(params.Handles) > 0 {
			targets = strings.Join(params.Handles, ", ")
		}
		if a.kind == agent.AgentCallCancel {
			return []string{targets}, true
		}
		return []string{params.Message, "to", targets}, true
	}, nil)
}
