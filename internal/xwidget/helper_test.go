package xwidget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestShowPopUpMenuBelowLeading_NilMenuIsNoop(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	lbl := widget.NewLabel("x")
	w := test.NewWindow(lbl)
	defer w.Close()

	assert.NotPanics(t, func() {
		xwidget.ShowPopUpMenuBelowLeading(lbl, nil)
	})
}

func TestShowPopUpMenuBelowLeading_ShowsMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	lbl := widget.NewLabel("x")
	w := test.NewWindow(lbl)
	defer w.Close()

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	assert.NotPanics(t, func() {
		xwidget.ShowPopUpMenuBelowLeading(lbl, menu)
	})
}

func TestShowPopUpMenuBelowTrailing_NilMenuIsNoop(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	lbl := widget.NewLabel("x")
	w := test.NewWindow(lbl)
	defer w.Close()

	assert.NotPanics(t, func() {
		xwidget.ShowPopUpMenuBelowTrailing(lbl, nil)
	})
}

func TestShowPopUpMenuBelowTrailing_ShowsMenu(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	lbl := widget.NewLabel("x")
	w := test.NewWindow(lbl)
	defer w.Close()

	menu := fyne.NewMenu("", fyne.NewMenuItem("Item1", nil))
	assert.NotPanics(t, func() {
		xwidget.ShowPopUpMenuBelowTrailing(lbl, menu)
	})
}
