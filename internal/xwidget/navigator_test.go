package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNavigator_PanicsWithoutRootAppBar(t *testing.T) {
	assert.Panics(t, func() {
		xwidget.NewNavigator(nil)
	})
}

func TestNavigator_PushAndPop(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	w := test.NewWindow(nav)
	defer w.Close()

	assert.True(t, nav.IsRoot())
	assert.False(t, nav.IsEmpty())
	assert.Equal(t, fyne.CanvasObject(root), nav.Current())

	second := xwidget.NewAppBar("Second", widget.NewLabel("Second"))
	nav.Push(second)

	assert.False(t, nav.IsRoot())
	assert.Equal(t, fyne.CanvasObject(second), nav.Current())

	popped := nav.Pop()
	assert.Equal(t, fyne.CanvasObject(second), popped)
	assert.True(t, nav.IsRoot())
}

func TestNavigator_PopOnRootPageReturnsNil(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	w := test.NewWindow(nav)
	defer w.Close()

	assert.Nil(t, nav.Pop())
	assert.True(t, nav.IsRoot())
}

func TestNavigator_PopAllRemovesEveryNonRootPage(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	w := test.NewWindow(nav)
	defer w.Close()

	nav.Push(xwidget.NewAppBar("Second", widget.NewLabel("Second")))
	nav.Push(xwidget.NewAppBar("Third", widget.NewLabel("Third")))

	nav.PopAll()

	assert.True(t, nav.IsRoot())
	assert.Equal(t, fyne.CanvasObject(root), nav.Current())
}

func TestNavigator_SetReplacesRootAndClearsStack(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	w := test.NewWindow(nav)
	defer w.Close()

	nav.Push(xwidget.NewAppBar("Second", widget.NewLabel("Second")))

	newRoot := xwidget.NewAppBar("New Root", widget.NewLabel("New Root"))
	nav.Set(newRoot)

	assert.True(t, nav.IsRoot())
	assert.Equal(t, fyne.CanvasObject(newRoot), nav.Current())
}

func TestNavigator_PushAndHideNavBarThenPopShowsBarAgain(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	root := xwidget.NewAppBar("Root", widget.NewLabel("Root"))
	nav := xwidget.NewNavigator(root)
	dest := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	nav.NavBar = xwidget.NewNavBar(dest)
	w := test.NewWindow(nav)
	defer w.Close()

	second := xwidget.NewAppBar("Second", widget.NewLabel("Second"))
	assert.NotPanics(t, func() {
		nav.PushAndHideNavBar(second)
	})
	assert.NotPanics(t, func() {
		nav.Pop()
	})
}
