package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNavList_CanCreate(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	it1 := xwidget.NewNavListItem("Alpha", theme.HomeIcon(), nil)
	it2 := xwidget.NewNavListItem("Bravo", theme.HomeIcon(), nil)
	list := xwidget.NewNavList(it1, it2)
	w := test.NewWindow(list)
	defer w.Close()

	assert.NotNil(t, list)
}

func TestNavListItem_TapRunsAction(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped bool
	it := xwidget.NewNavListItem("Headline", theme.HomeIcon(), func() {
		tapped = true
	})
	w := test.NewWindow(it)
	defer w.Close()

	test.Tap(it)
	assert.True(t, tapped)
}

func TestNavListItem_TapWhileDisabledDoesNothing(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var tapped bool
	it := xwidget.NewNavListItem("Headline", theme.HomeIcon(), func() {
		tapped = true
	})
	it.IsDisabled = true
	w := test.NewWindow(it)
	defer w.Close()

	test.Tap(it)
	assert.False(t, tapped)
}

func TestNavListItem_RefreshAfterFieldChanges(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	it := xwidget.NewNavListItem("Headline", theme.HomeIcon(), nil)
	w := test.NewWindow(it)
	defer w.Close()

	it.Headline = "New Headline"
	it.Supporting = "Supporting text"
	it.Leading = theme.CancelIcon()
	it.Trailing = theme.CancelIcon()
	it.IsDisabled = true

	assert.NotPanics(t, it.Refresh)
}
