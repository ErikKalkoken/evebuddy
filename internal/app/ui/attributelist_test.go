package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestAttributeList(t *testing.T) {
	test.NewTempApp(t)
	var infoTapped, rowTapped bool
	w := ui.NewAttributeList(
		ui.AttributeItem{Label: "Alpha", Value: "1", InfoAction: func() { infoTapped = true }},
		ui.AttributeItem{Label: "Bravo", Value: "2", Action: func() { rowTapped = true }},
	)
	win := test.NewWindow(w)
	defer win.Close()
	win.Resize(fyne.NewSize(300, 200))

	var icons []*xwidget.TappableIcon
	for _, o := range test.LaidOutObjects(w) {
		c, ok := o.(*fyne.Container)
		if !ok || !c.Visible() || len(c.Objects) != 3 {
			continue
		}
		if x, ok := c.Objects[1].(*xwidget.TappableIcon); ok {
			icons = append(icons, x)
		}
	}
	t.Run("should show info icon only for items with info action", func(t *testing.T) {
		assert.Len(t, icons, 1)
	})
	t.Run("should call info action when tapping the icon", func(t *testing.T) {
		test.Tap(icons[0])
		assert.True(t, infoTapped)
	})
	t.Run("should call action when tapping the row", func(t *testing.T) {
		list := test.WidgetRenderer(w).Objects()[0].(*widget.List)
		list.Select(1)
		assert.True(t, rowTapped)
	})
}
