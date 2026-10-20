package model

// Queued-prompt transcript entry.
//
// A prompt submitted while the agent is busy is enqueued server-side and
// does not exist as a message yet. The transcript nevertheless shows it
// the moment it is entered: an optimistic placeholder is appended at
// send time, and the authoritative queue (fetched off-thread, see
// workspace_cache.go) reconciles it. Multiple queued prompts join into
// the single placeholder — mirroring the single user message the agent
// creates when the queue drains — so the swap from placeholder to real
// message happens in one update pass and is invisible. Prompts queued
// from another client join the same entry through the reconcile.
//
// The queue belongs to the main session, so the placeholder paints only
// into the main transcript: while an agent's session is on screen the
// prompts are tracked but not drawn, and the main transcript re-adds
// the placeholder when it comes back.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stubbedev/harness/internal/message"
	"github.com/stubbedev/harness/internal/ui/chat"
)

// appendQueuedPrompt adds text to the transcript's queued-prompt
// placeholder, joining it onto the prompts already shown.
func (m *UI) appendQueuedPrompt(text string) {
	if text == "" || m.chat == nil {
		return
	}
	m.queuedPromptsSession = m.currentSessionID()
	m.queuedPrompts = append(m.queuedPrompts, text)
	m.syncQueuedPromptItem()
}

// syncQueuedPromptItem aligns the single placeholder item with
// m.queuedPrompts: it is created on the first queued prompt, re-textured
// as more join, and dropped when the last one leaves. It never touches
// a transcript other than the main one of the session the prompts were
// queued on.
func (m *UI) syncQueuedPromptItem() {
	if m.agentView.shown != "" || m.queuedPromptsSession != m.currentSessionID() {
		return
	}
	if len(m.queuedPrompts) == 0 {
		if m.queuedPromptItem != nil {
			m.chat.RemoveMessage(m.queuedPromptItem.ID())
			m.queuedPromptItem = nil
		}
		return
	}
	if m.queuedPromptItem == nil {
		m.queuedPromptSeq++
		m.queuedPromptItem = chat.NewQueuedMessageItem(m.com.Styles, fmt.Sprintf("queued-prompt-%d", m.queuedPromptSeq), m.queuedPrompts)
		m.chat.AppendMessages(m.queuedPromptItem)
	} else {
		m.queuedPromptItem.UpdateTexts(m.queuedPrompts)
	}
	m.chat.ScrollToBottom()
}

// materializeQueuedPrompt consumes the queued prompts that became the
// user message that just landed in the transcript: the drained queue
// materializes as a single message joining every queued prompt, and a
// queued call that ran as its own turn (RunID path) as one segment.
func (m *UI) materializeQueuedPrompt(text string) {
	if text == "" || m.chat == nil || len(m.queuedPrompts) == 0 {
		return
	}
	consumed := 0
	for i := range m.queuedPrompts {
		if strings.Join(m.queuedPrompts[:i+1], message.QueuedPromptSeparator) == text {
			consumed = i + 1
			break
		}
	}
	if consumed == 0 {
		return
	}
	m.queuedPrompts = m.queuedPrompts[consumed:]
	m.syncQueuedPromptItem()
}

// reconcileQueuedPrompts aligns the placeholder with the authoritative
// queue list: prompts that left the queue without a matching user
// message (cleared, or drained through a path that never publishes one)
// drop out, and prompts not yet shown — e.g. queued from another client
// — join the entry.
func (m *UI) reconcileQueuedPrompts(prompts []string) {
	if m.chat == nil {
		return
	}
	m.queuedPromptsSession = m.currentSessionID()
	m.queuedPrompts = slices.Clone(prompts)
	m.syncQueuedPromptItem()
}

// resetQueuedPromptsUnlessFor runs when the transcript list is replaced:
// the replacement drops the placeholder item, so it is forgotten, and
// the queued prompts are dropped too unless they belong to sessionID,
// the session whose main transcript is being rebuilt.
func (m *UI) resetQueuedPromptsUnlessFor(sessionID string) {
	m.queuedPromptItem = nil
	if m.queuedPromptsSession != sessionID {
		m.queuedPrompts = nil
		m.queuedPromptsSession = sessionID
	}
}
