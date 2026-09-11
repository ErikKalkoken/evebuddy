package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestAppBar_CanCreateAndSetTitle(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	body := widget.NewLabel("Body")
	ab := xwidget.NewAppBar("Title", body)
	w := test.NewWindow(ab)
	defer w.Close()

	assert.Equal(t, "Title", ab.Title())

	ab.SetTitle("New Title")
	assert.Equal(t, "New Title", ab.Title())
}

func TestAppBar_CanCreateWithTrailingItems(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	body := widget.NewLabel("Body")
	trailing := widget.NewButton("Action", nil)
	ab := xwidget.NewAppBar("Title", body, trailing)
	w := test.NewWindow(ab)
	defer w.Close()

	assert.NotPanics(t, ab.Refresh)
}

func TestAppBar_CanCreateWithHiddenBackground(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	ab := xwidget.NewAppBar("Title", widget.NewLabel("Body"))
	ab.HideBackground = true
	w := test.NewWindow(ab)
	defer w.Close()

	assert.NotPanics(t, ab.Refresh)
}

func TestAppBar_WhenPushedItGetsANavigator(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	second := xwidget.NewAppBar("Second", widget.NewLabel("Second"))
	nav.Push(second)
	w := test.NewWindow(nav)
	defer w.Close()

	assert.Equal(t, nav, second.Navigator)
	assert.NotPanics(t, func() {
		second.Navigator.Pop()
	})
}
