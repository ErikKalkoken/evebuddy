package widget_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

func TestStrippedList_Create(t *testing.T) {
	test.NewTempApp(t)
	test.ApplyTheme(t, test.Theme())

	data := []string{"alpha", "bravo", "charlie"}
	list := iwidget.NewStripedList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(data[i])
		})

	w := test.NewWindow(list)
	defer w.Close()
	w.Resize(fyne.NewSize(500, 500))

	test.AssertImageMatches(t, "strippedlist/master.png", w.Canvas().Capture())
}
