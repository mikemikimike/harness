package agent

import (
	"testing"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
	"github.com/stubbedev/harness/internal/catalog"
	"github.com/stubbedev/harness/internal/session"
)

// titleAfterFirstStep records a first step of 100 prompt and 20 completion
// tokens costing 0.5, then titles the session with a small model priced at
// 2 per 1M input and 4 per 1M output tokens that reports 1000 input and 50
// output tokens, as a title finishing after step 1 would.
func titleAfterFirstStep(t *testing.T) session.Session {
	t.Helper()

	env := testEnv(t)
	small := textModel("Priced Title")
	small.usage = fantasy.Usage{InputTokens: 1000, OutputTokens: 50, TotalTokens: 1050}
	sa := NewSessionAgent(SessionAgentOptions{
		LargeModel: Model{
			Model:      textModel("unused"),
			CatalogCfg: catalog.Model{ContextWindow: 200000, DefaultMaxTokens: 10000},
		},
		SmallModel: Model{
			Model: small,
			CatalogCfg: catalog.Model{
				ContextWindow:    200000,
				DefaultMaxTokens: 10000,
				CostPer1MIn:      2,
				CostPer1MOut:     4,
			},
		},
		Sessions: env.sessions,
		Messages: env.messages,
	})

	sess, err := env.sessions.Create(t.Context(), "New Session")
	require.NoError(t, err)
	estimated := false
	_, err = env.sessions.RecordUsage(t.Context(), sess.ID, session.Usage{
		CostDelta:        0.5,
		PromptTokens:     100,
		CompletionTokens: 20,
		Estimated:        &estimated,
	})
	require.NoError(t, err)

	sa.GenerateTitle(t.Context(), sess.ID, "add a priced title")

	got, err := env.sessions.Get(t.Context(), sess.ID)
	require.NoError(t, err)
	require.Equal(t, "Priced Title", got.Title)
	return got
}

func TestGenerateTitleChargesItsCost(t *testing.T) {
	t.Parallel()

	got := titleAfterFirstStep(t)
	require.InDelta(t, 0.5+2.0*1000/1e6+4.0*50/1e6, got.Cost, 1e-12)
}

// The token counters hold the context size of the last coding step, so a
// title finishing after step 1 must leave them at step 1's.
func TestGenerateTitleLeavesTokenCountersAlone(t *testing.T) {
	t.Parallel()

	got := titleAfterFirstStep(t)
	require.Equal(t, int64(100), got.PromptTokens)
	require.Equal(t, int64(20), got.CompletionTokens)
}
