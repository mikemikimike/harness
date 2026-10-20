package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
	"github.com/stubbedev/harness/internal/ui/keys"
)

// newShortcutPalette opens the palette inside a session, so the
// session-only shortcuts are listed too.
func newShortcutPalette(t *testing.T) *Commands {
	t.Helper()

	com := paletteTestCommon()
	c, err := NewCommands(com, NewCatalog(com, CommandState{SessionID: "s"}, nil, nil))
	require.NoError(t, err)
	return c
}

// shortcutKeyPress builds the key press whose String() is k, for the
// modifier chords palette shortcuts are bound to.
func shortcutKeyPress(t *testing.T, k string) tea.KeyPressMsg {
	t.Helper()

	parts := strings.Split(k, "+")
	var msg tea.KeyPressMsg
	for _, mod := range parts[:len(parts)-1] {
		switch mod {
		case "ctrl":
			msg.Mod |= tea.ModCtrl
		case "shift":
			msg.Mod |= tea.ModShift
		default:
			t.Fatalf("unsupported modifier %q in %q", mod, k)
		}
	}
	code := []rune(parts[len(parts)-1])
	require.Len(t, code, 1, "unsupported key %q", k)
	msg.Code = code[0]
	require.Equal(t, k, msg.String())
	return msg
}

// paletteItems returns the command items the palette currently lists.
func paletteItems(c *Commands) []*CommandItem {
	var items []*CommandItem
	for _, it := range c.list.FilteredItems() {
		if item, ok := it.(*CommandItem); ok {
			items = append(items, item)
		}
	}
	return items
}

// TestPaletteShortcutFiresOnEveryKey pins that a shortcut is matched as
// a binding, not as its label: every key of a multi-key binding (Switch
// Model is ctrl+m and ctrl+l) selects its entry.
func TestPaletteShortcutFiresOnEveryKey(t *testing.T) {
	t.Parallel()

	c := newShortcutPalette(t)
	var fired int
	for _, item := range paletteItems(c) {
		for _, k := range item.Shortcut().Keys() {
			action := c.HandleMsg(shortcutKeyPress(t, k))
			require.Equal(t, item.SelectAction(), action, "%s on %q", k, item.Title())
			fired++
		}
	}
	require.Positive(t, fired, "the palette lists no shortcuts")
}

// TestPaletteSwitchModelFiresOnUnlabelledKey is the regression for the
// label comparison: ctrl+m opens the model picker even though the row
// is hinted with ctrl+l.
func TestPaletteSwitchModelFiresOnUnlabelledKey(t *testing.T) {
	t.Parallel()

	c := newShortcutPalette(t)
	action := c.HandleMsg(shortcutKeyPress(t, "ctrl+m"))
	invoke, ok := action.(ActionInvoke)
	require.True(t, ok, "ctrl+m yielded %T", action)
	require.Equal(t, ActionOpenDialog{ModelsID}, invoke.Action)
}

// TestPaletteNeverHintsAConsumedKey pins that no entry is hinted with,
// or bound to, a key the palette's own navigation handles first, and
// that a hint always names a key the entry fires on.
func TestPaletteNeverHintsAConsumedKey(t *testing.T) {
	t.Parallel()

	c := newShortcutPalette(t)
	consumed := map[string]bool{}
	for _, b := range c.ownKeys() {
		for _, k := range b.Keys() {
			consumed[k] = true
		}
	}
	for _, item := range paletteItems(c) {
		for _, k := range item.Shortcut().Keys() {
			require.False(t, consumed[k], "%q is bound to %s, which the palette consumes", item.Title(), k)
		}
		if label := item.InfoText(); label != "" {
			require.Contains(t, item.Shortcut().Keys(), label, "%q is hinted with a key it does not fire on", item.Title())
		}
	}
}

// TestPaletteNewSessionKeyNavigates is the regression for the ctrl+n
// collision: ctrl+n is the palette's next-item key, so it moves the
// selection and New Session is not hinted with it.
func TestPaletteNewSessionKeyNavigates(t *testing.T) {
	t.Parallel()

	c := newShortcutPalette(t)
	for _, item := range paletteItems(c) {
		if item.ID() == "new_session" {
			require.Empty(t, item.InfoText())
		}
	}
	before := c.list.Selected()
	require.Nil(t, c.HandleMsg(shortcutKeyPress(t, "ctrl+n")))
	require.NotEqual(t, before, c.list.Selected())
}

// TestPaletteKeepsDefaultShortcutHints pins that stripping consumed keys
// leaves every other default hint in place.
func TestPaletteKeepsDefaultShortcutHints(t *testing.T) {
	t.Parallel()

	km := keys.DefaultKeyMap()
	want := map[string]string{
		"switch_session":      km.Sessions.Help().Key,
		"switch_model":        km.Models.Help().Key,
		"switch_theme":        km.Themes.Help().Key,
		"export_conversation": km.ExportConversation.Help().Key,
		"toggle_help":         km.Help.Help().Key,
		"quit":                km.Quit.Help().Key,
	}
	c := newShortcutPalette(t)
	for _, item := range paletteItems(c) {
		if label, ok := want[item.ID()]; ok {
			require.Equal(t, label, item.InfoText(), item.ID())
			delete(want, item.ID())
		}
	}
	require.Empty(t, want, "entries missing from the palette")
}
