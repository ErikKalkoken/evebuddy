package desktopui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"

	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

// NewConfirmDialog returns a new pre-configured confirm dialog.
func NewConfirmDialog(title, message, confirm string, callback func(bool), parent fyne.Window) *dialog.ConfirmDialog {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	kxdialog.AddDialogKeyHandler(d, parent)
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
	kxdialog.AddDialogKeyHandler(d, parent)
	return d
}
