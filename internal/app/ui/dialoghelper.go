package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/humanize"
)

// NewConfirmDialog returns a new custom confirm dialog.
func NewConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) *dialog.ConfirmDialog {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	AddDialogKeyHandler(d, parent)
	return d
}

// NewErrorDialog returns a new custom error dialog.
func NewErrorDialog(message string, err error, parent fyne.Window) dialog.Dialog {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, parent)
	AddDialogKeyHandler(d, parent)
	return d
}

// AddDialogKeyHandler adds a minimal key handler to a dialog.
// It enables the user to close the dialog by pressing the escape key.
//
// Note that previously defined key events will be shadowed while the dialog is open.
func AddDialogKeyHandler(d dialog.Dialog, w fyne.Window) {
	addDialogKeyHandler(d, w, func(ke *fyne.KeyEvent, d dialog.Dialog) {
		if ke.Name == fyne.KeyEscape {
			d.Hide()
		}
	})
}

// AddConfirmDialogKeyHandler adds a key handler to a confirm dialog.
// It enables the user to confirm the dialog by pressing the return key
// and close the dialog when the user presses the escape key.
//
// The callback will be called when the user presses the return key instead of the dialog callback.
//
// Note that previously defined key events will be shadowed while the dialog is open.
func AddConfirmDialogKeyHandler(d *dialog.ConfirmDialog, callback func(bool), w fyne.Window) {
	addDialogKeyHandler(d, w, func(ke *fyne.KeyEvent, d dialog.Dialog) {
		switch ke.Name {
		case fyne.KeyEscape:
			d.Hide()
			callback(false)
		case fyne.KeyReturn, fyne.KeyEnter:
			d.Hide()
			callback(true)
		}
	})
}

func addDialogKeyHandler(d dialog.Dialog, w fyne.Window, handler func(ke *fyne.KeyEvent, d dialog.Dialog)) {
	originalEvent := w.Canvas().OnTypedKey() // == nil when not set
	w.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		if d == nil {
			return
		}
		handler(ke, d)
	})
	d.SetOnClosed(func() {
		w.Canvas().SetOnTypedKey(originalEvent)
	})
}
