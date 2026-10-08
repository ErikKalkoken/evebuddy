package ui_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestAttributeList(t *testing.T) {
	test.NewTempApp(t)
	var infoTapped, rowTapped, headingTapped bool
	w := ui.NewAttributeList(
		ui.AttributeItem{Label: "Alpha", Value: "1", InfoAction: func() { infoTapped = true }},
		ui.AttributeItem{Label: "Bravo", Value: "2", Action: func() { rowTapped = true }},
		ui.AttributeItem{
			Label:      "Charlie",
			Value:      "3",
			IsHeading:  true,
			Action:     func() { headingTapped = true },
			InfoAction: func() {},
		},
	)
	win := test.NewWindow(w)
	defer win.Close()
	win.Resize(fyne.NewSize(300, 200))

	var icons []*xwidget.IconButton
	for _, o := range test.LaidOutObjects(w) {
		c, ok := o.(*fyne.Container)
		if !ok || !c.Visible() || len(c.Objects) != 3 {
			continue
		}
		if x, ok := c.Objects[1].(*xwidget.IconButton); ok {
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
	t.Run("should show heading in bold without value", func(t *testing.T) {
		var heading *widget.Label
		var values []string
		for _, o := range test.LaidOutObjects(w) {
			x, ok := o.(*widget.Label)
			if !ok || !x.Visible() {
				continue
			}
			if x.Text == "Charlie" {
				heading = x
			}
			values = append(values, x.Text)
		}
		require.NotNil(t, heading)
		assert.True(t, heading.TextStyle.Bold)
		assert.NotContains(t, values, "3")
	})
	t.Run("should not call action when tapping a heading", func(t *testing.T) {
		list := test.WidgetRenderer(w).Objects()[0].(*widget.List)
		list.Select(2)
		assert.False(t, headingTapped)
	})
}
