package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
)

// newConfirmDialog returns a new custom confirm dialog.
func newConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) *dialog.ConfirmDialog {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	return d
}

// newErrorDialog returns a new custom error dialog.
func newErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, parent)
	return d
}

// showProgressModal shows a modal with a progress indicator while an action is running.
func showProgressModal(action, success, failure string, f func() error, parent fyne.Window) {
	pg := widget.NewProgressBarInfinite()
	pg.Start()
	d1 := dialog.NewCustomWithoutButtons(action, pg, parent)
	d1.Show()
	go func() {
		err := f()
		d1.Hide()
		if err != nil {
			d2 := newErrorDialog(failure, err, parent)
			d2.Show()
		} else {
			d2 := dialog.NewInformation("Completed", success, parent)
			d2.Show()
		}
	}()
}
