package agent

import (
	"encoding/json"

	"charm.land/fantasy"

	"github.com/stubbedev/harness/internal/agent/tools"
)

// earlyDispatchTools are the tools whose calls may start while the model is
// still streaming the rest of its message (fantasy.WithEarlyToolDispatch).
//
// Being Parallel is not enough to be listed: edit, write and shell are
// Parallel too. If the step then ends on anything but a tool-calls finish,
// an early call is canceled and its result discarded, exactly like a call
// that never ran, but it may already have run by then. So only tools that
// read, and whose side effects (a file marked as seen, a PreToolUse hook, a
// spilled fetch result) are harmless for a call whose result the model
// never saw, belong here.
var earlyDispatchTools = map[string]bool{
	tools.ViewToolName:      true,
	tools.FetchToolName:     true,
	tools.WebSearchToolName: true,
	tools.HarnessToolName:   true,
}

// earlyToolDispatch reports whether a call may start before the stream it
// arrived in has ended. fantasy only asks about calls to Parallel tools
// whose arguments are already valid.
//
// The lsp tool is listed by action: its questions (diagnostics, symbols,
// definition, references, call_hierarchy) qualify, its rename,
// replace_symbol and restart do not.
//
// Sub-agent dispatches start early too, so the first agent of a fan-out
// runs while the model is still writing the prompts of the rest instead
// of waiting for the whole message. A dispatch the step then discards is
// cancelled and forgotten (see SubagentDispatchDiscarder). The
// dispatcher's wait, message and cancel forms act on agents already
// running and wait for the end of the stream like any other call.
func earlyToolDispatch(call fantasy.ToolCallContent) bool {
	switch call.ToolName {
	case tools.LSPToolName:
		return tools.LSPCallReadOnly(call.Input)
	case AgentToolName:
		var params AgentDispatchParams
		return json.Unmarshal([]byte(call.Input), &params) == nil &&
			params.validateForm() == nil &&
			params.Kind() == AgentCallDispatch
	case tools.ResearchToolName:
		return true
	}
	return earlyDispatchTools[call.ToolName]
}

// isSubagentDispatchTool reports whether a tool's calls start sub-agent
// runs.
func isSubagentDispatchTool(name string) bool {
	return name == AgentToolName || name == tools.ResearchToolName
}
