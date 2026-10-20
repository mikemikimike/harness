package dialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
	"github.com/stubbedev/harness/internal/catalog"
	"github.com/stubbedev/harness/internal/config"
	"github.com/stubbedev/harness/internal/ui/common"
	"github.com/stubbedev/harness/internal/ui/styles"
)

func newAPIKeyInputForTest(t *testing.T) *APIKeyInput {
	t.Helper()
	s := styles.CharmtonePantera()
	com := &common.Common{Workspace: &stubWorkspace{cfg: &config.Config{}}, Styles: &s}
	d, _ := NewAPIKeyInput(com, catalog.Provider{ID: "anthropic", Name: "Anthropic"}, config.SelectedModel{}, config.SelectedModelTypeLarge)
	return d
}

func typeAPIKey(d *APIKeyInput, text string) {
	for _, r := range text {
		d.HandleMsg(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func requireCursorAfter(t *testing.T, cur *tea.Cursor, rows []string, typed string) {
	t.Helper()
	require.NotNil(t, cur)
	row := ansi.Strip(rows[cur.Y])
	end := strings.Index(row, typed)
	require.NotEqual(t, -1, end, "cursor row %d does not hold the typed text %q:\n%s", cur.Y, typed, strings.Join(rows, "\n"))
	require.Equal(t, ansi.StringWidth(row[:end])+ansi.StringWidth(typed), cur.X,
		"cursor must sit on the cell right after the typed text:\n%s", strings.Join(rows, "\n"))
}

func assertAPIKeyCursorPlacement(t *testing.T) {
	t.Helper()

	t.Run("end of key", func(t *testing.T) {
		d := newAPIKeyInputForTest(t)
		drawRows(t, d, 100, 30)
		typeAPIKey(d, "sk-test")
		cur, rows := drawRows(t, d, 100, 30)
		requireCursorAfter(t, cur, rows, "sk-test")
	})

	t.Run("after moving left", func(t *testing.T) {
		d := newAPIKeyInputForTest(t)
		drawRows(t, d, 100, 30)
		typeAPIKey(d, "sk-test")
		for range 4 {
			d.HandleMsg(tea.KeyPressMsg{Code: tea.KeyLeft})
		}
		cur, rows := drawRows(t, d, 100, 30)
		requireCursorAfter(t, cur, rows, "sk-")
	})

	t.Run("long key scrolled horizontally", func(t *testing.T) {
		d := newAPIKeyInputForTest(t)
		drawRows(t, d, 100, 30)
		key := "sk-" + strings.Repeat("abcdefghij", 20) + "-END"
		typeAPIKey(d, key)
		cur, rows := drawRows(t, d, 100, 30)
		requireCursorAfter(t, cur, rows, "-END")
	})
}

func TestAPIKeyInputCursorPlacementBottomAnchored(t *testing.T) {
	t.Cleanup(func() { InstallPlacement(config.DialogPlacementBottom) })
	InstallPlacement(config.DialogPlacementBottom)
	assertAPIKeyCursorPlacement(t)
}

func TestAPIKeyInputCursorPlacementTopFloating(t *testing.T) {
	t.Cleanup(func() { InstallPlacement(config.DialogPlacementBottom) })
	InstallPlacement(config.DialogPlacementTop)
	assertAPIKeyCursorPlacement(t)
}

func assertArgumentsCursorPlacement(t *testing.T) {
	t.Helper()
	d := newTestArguments()
	drawRows(t, d, 100, 30)
	typeIntoArguments(d, "focusword")
	cur, rows := drawRows(t, d, 100, 30)
	requireCursorAfter(t, cur, rows, "focusword")
}

func TestArgumentsCursorPlacementBottomAnchored(t *testing.T) {
	t.Cleanup(func() { InstallPlacement(config.DialogPlacementBottom) })
	InstallPlacement(config.DialogPlacementBottom)
	assertArgumentsCursorPlacement(t)
}

func TestArgumentsCursorPlacementTopFloating(t *testing.T) {
	t.Cleanup(func() { InstallPlacement(config.DialogPlacementBottom) })
	InstallPlacement(config.DialogPlacementTop)
	assertArgumentsCursorPlacement(t)
}
