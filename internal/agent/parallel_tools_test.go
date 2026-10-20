package agent

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"

	"github.com/stubbedev/harness/internal/agent/tools"
	"github.com/stubbedev/harness/internal/catalog"
	"github.com/stubbedev/harness/internal/config"
	"github.com/stubbedev/harness/internal/session"
)

// probeTool blocks each call until `want` calls have started, so a test
// can tell whether the calls of one step ran together or one after the
// other. Sequential execution never reaches `want` started calls at once
// and times out into a plain result instead of deadlocking.
type probeTool struct {
	fantasy.AgentTool
	mu      sync.Mutex
	started int
	peak    atomic.Int32
	inside  atomic.Int32
	all     chan struct{}
	want    int
}

func newProbeTool(t *testing.T, want int, parallel bool) *probeTool {
	t.Helper()
	p := &probeTool{all: make(chan struct{}), want: want}
	type args struct {
		N int `json:"n"`
	}
	run := func(ctx context.Context, in args, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
		now := p.inside.Add(1)
		defer p.inside.Add(-1)
		for {
			prev := p.peak.Load()
			if now <= prev || p.peak.CompareAndSwap(prev, now) {
				break
			}
		}
		p.mu.Lock()
		p.started++
		if p.started == p.want {
			close(p.all)
		}
		p.mu.Unlock()
		select {
		case <-p.all:
		case <-time.After(300 * time.Millisecond):
		case <-ctx.Done():
		}
		return fantasy.NewTextResponse("probed"), nil
	}
	if parallel {
		p.AgentTool = fantasy.NewParallelAgentTool("probe", "probe", run)
	} else {
		p.AgentTool = fantasy.NewAgentTool("probe", "probe", run)
	}
	return p
}

func probeAgent(t *testing.T, env fakeEnv, probe fantasy.AgentTool) SessionAgent {
	t.Helper()
	model := newScriptedModel(scriptedTurn{calls: []scriptedCall{
		{name: "probe", input: map[string]any{"n": 1}},
		{name: "probe", input: map[string]any{"n": 2}},
		{name: "probe", input: map[string]any{"n": 3}},
	}})
	return NewSessionAgent(SessionAgentOptions{
		LargeModel: Model{
			Model:      model,
			CatalogCfg: catalog.Model{ID: "large", ContextWindow: 1_000_000, DefaultMaxTokens: 1000},
			ModelCfg:   config.SelectedModel{Model: "large", Provider: "scripted"},
		},
		SmallModel:   Model{Model: textModel("title")},
		SystemPrompt: "system",
		Sessions:     env.sessions,
		Messages:     env.messages,
		Tools:        []fantasy.AgentTool{probe},
	})
}

// Read-only tools a step calls together run together: three parallel
// probes are all in flight at once.
func TestParallelToolsRunTogetherWithinAStep(t *testing.T) {
	env := testEnv(t)
	probe := newProbeTool(t, 3, true)
	sess, err := env.sessions.Create(t.Context(), "parallel")
	require.NoError(t, err)

	_, err = probeAgent(t, env, probe).Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "probe"})
	require.NoError(t, err)
	require.Equal(t, int32(3), probe.peak.Load(), "all three calls must overlap")
}

// Tools not marked parallel keep running one at a time, in order.
func TestSequentialToolsRunOneAtATime(t *testing.T) {
	env := testEnv(t)
	probe := newProbeTool(t, 3, false)
	sess, err := env.sessions.Create(t.Context(), "sequential")
	require.NoError(t, err)

	_, err = probeAgent(t, env, probe).Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "probe"})
	require.NoError(t, err)
	require.Equal(t, int32(1), probe.peak.Load(), "sequential tools never overlap")
}

type recordingDiscarder struct {
	mu       sync.Mutex
	children []string
}

func (d *recordingDiscarder) DiscardSubagentDispatch(childSessionID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.children = append(d.children, childSessionID)
}

func (d *recordingDiscarder) discarded() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.children...)
}

// A sub-agent dispatch starts while its message is still streaming; when
// the step then ends abnormally the session agent hands the dispatch's
// child session to the discarder, so the run the model never saw the
// handle of is undone. A completed step discards nothing.
func TestEarlySubagentDispatchDiscardedWithItsStep(t *testing.T) {
	for _, tc := range []struct {
		name    string
		finish  fantasy.FinishReason
		wantOne bool
	}{
		{name: "length", finish: fantasy.FinishReasonLength, wantOne: true},
		{name: "tool calls", finish: fantasy.FinishReasonToolCalls},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := testEnv(t)
			started := make(chan string, 1)
			dispatch := fantasy.NewParallelAgentTool(AgentToolName, "dispatch",
				func(ctx context.Context, in AgentDispatchParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
					messageID, _ := ctx.Value(tools.MessageIDContextKey).(string)
					started <- messageID
					return fantasy.NewTextResponse("handle bg-1"), nil
				})
			discarder := &recordingDiscarder{}
			model := newScriptedModel(scriptedTurn{
				calls:  []scriptedCall{{name: AgentToolName, input: map[string]any{"prompt": "audit"}}},
				finish: tc.finish,
			})
			a := NewSessionAgent(SessionAgentOptions{
				LargeModel: Model{
					Model:      model,
					CatalogCfg: catalog.Model{ID: "large", ContextWindow: 1_000_000, DefaultMaxTokens: 1000},
					ModelCfg:   config.SelectedModel{Model: "large", Provider: "scripted"},
				},
				SmallModel:      Model{Model: textModel("title")},
				SystemPrompt:    "system",
				Sessions:        env.sessions,
				Messages:        env.messages,
				Tools:           []fantasy.AgentTool{dispatch},
				SubagentDiscard: discarder,
			})
			sess, err := env.sessions.Create(t.Context(), "early")
			require.NoError(t, err)

			_, _ = a.Run(t.Context(), SessionAgentCall{SessionID: sess.ID, Prompt: "fan out"})
			require.Len(t, started, 1, "the dispatch started")
			dispatchMessageID := <-started

			if !tc.wantOne {
				require.Empty(t, discarder.discarded())
				return
			}
			discarded := discarder.discarded()
			require.Len(t, discarded, 1)
			messageID, toolCallID, ok := session.ParseAgentToolSessionID(discarded[0])
			require.True(t, ok, "the discarder gets the dispatch's child session ID")
			require.Equal(t, dispatchMessageID, messageID, "the child session is the one the dispatch created")
			require.Equal(t, "call-1-0", toolCallID)
		})
	}
}
