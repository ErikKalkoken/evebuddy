package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestNavBar_PanicsWithoutDestinations(t *testing.T) {
	assert.Panics(t, func() {
		xwidget.NewNavBar()
	})
}

func TestNavBar_SelectsFirstDestinationOnCreation(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selectedA int
	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	dA.OnSelected = func() { selectedA++ }
	dB := xwidget.NewDestinationDef("B", theme.HomeIcon(), widget.NewLabel("B"))

	nb := xwidget.NewNavBar(dA, dB)
	w := test.NewWindow(nb)
	defer w.Close()

	id, ok := nb.Selected()
	assert.True(t, ok)
	assert.Equal(t, 0, id)
	assert.Equal(t, 1, selectedA)
}

func TestNavBar_SelectSwitchesDestinationAndFiresCallback(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selectedB int
	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	dB := xwidget.NewDestinationDef("B", theme.HomeIcon(), widget.NewLabel("B"))
	dB.OnSelected = func() { selectedB++ }

	nb := xwidget.NewNavBar(dA, dB)
	w := test.NewWindow(nb)
	defer w.Close()

	nb.Select(1)
	id, ok := nb.Selected()
	assert.True(t, ok)
	assert.Equal(t, 1, id)
	assert.Equal(t, 1, selectedB)
}

func TestNavBar_SelectingCurrentDestinationAgainFiresOnSelectedAgain(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	var selectedAgain int
	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	dA.OnSelectedAgain = func() { selectedAgain++ }

	nb := xwidget.NewNavBar(dA)
	w := test.NewWindow(nb)
	defer w.Close()

	nb.Select(0)
	assert.Equal(t, 1, selectedAgain)
}

func TestNavBar_EnableAndDisableDestination(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	dB := xwidget.NewDestinationDef("B", theme.HomeIcon(), widget.NewLabel("B"))
	nb := xwidget.NewNavBar(dA, dB)
	w := test.NewWindow(nb)
	defer w.Close()

	assert.True(t, nb.Enabled(1))

	nb.Disable(1)
	assert.False(t, nb.Enabled(1))

	nb.Select(1) // disabled destination must not be selectable
	id, _ := nb.Selected()
	assert.Equal(t, 0, id)

	nb.Enable(1)
	assert.True(t, nb.Enabled(1))

	nb.Select(1)
	id, _ = nb.Selected()
	assert.Equal(t, 1, id)
}

func TestNavBar_OutOfBoundsCallsAreNoop(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	nb := xwidget.NewNavBar(dA)
	w := test.NewWindow(nb)
	defer w.Close()

	assert.False(t, nb.Enabled(5))
	assert.False(t, nb.Enabled(-1))
	assert.NotPanics(t, func() { nb.Enable(5) })
	assert.NotPanics(t, func() { nb.Disable(-1) })
	assert.NotPanics(t, func() { nb.Select(99) })
	assert.NotPanics(t, func() { nb.ShowBadge(99, widget.HighImportance) })
	assert.NotPanics(t, func() { nb.HideBadge(99) })
}

func TestNavBar_ShowAndHideBadgeAndBar(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	dA := xwidget.NewDestinationDef("A", theme.HomeIcon(), widget.NewLabel("A"))
	nb := xwidget.NewNavBar(dA)
	w := test.NewWindow(nb)
	defer w.Close()

	assert.NotPanics(t, func() { nb.ShowBadge(0, widget.DangerImportance) })
	assert.NotPanics(t, func() { nb.HideBadge(0) })
	assert.NotPanics(t, nb.HideBar)
	assert.NotPanics(t, nb.ShowBar)
}
