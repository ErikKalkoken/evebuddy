package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNewToolbarActionMenu_ActivatingShowsMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	a := xwidget.NewToolbarActionMenu(theme.HomeIcon(), menu)
	tb := widget.NewToolbar(a)
	w := test.NewWindow(tb)
	defer w.Close()

	require.NotNil(t, a.OnActivated)
	assert.NotPanics(t, a.OnActivated)
}

func TestSetToolbarActionMenu_ReplacesActivationHandler(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	a := widget.NewToolbarAction(theme.HomeIcon(), nil)
	tb := widget.NewToolbar(a)
	w := test.NewWindow(tb)
	defer w.Close()

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	xwidget.SetToolbarActionMenu(a, menu)

	require.NotNil(t, a.OnActivated)
	assert.NotPanics(t, a.OnActivated)
}
