package ui

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
)

// ShowInformation shows a custom information dialog.
func ShowInformation(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	xdesktop.DisableShortcutsForDialog(d, parent)
	d.Show()
}

// ShowConfirm shows a custom confirmation dialog.
func ShowConfirm(title, message, confirm string, callback func(bool), parent fyne.Window) {
	d := dialog.NewConfirm(title, message, callback, parent)
	d.SetConfirmImportance(widget.DangerImportance)
	d.SetConfirmText(confirm)
	d.SetDismissText("Cancel")
	xdesktop.DisableShortcutsForDialog(d, parent)
	d.Show()
}

// ShowErrorAndLog shows a error dialog and logs the error.
func ShowErrorAndLog(message string, err error, IsDeveloperMode bool, parent fyne.Window) {
	slog.Error(message, "error", err)
	title := widget.NewLabel(message)
	var s string
	if IsDeveloperMode {
		s = err.Error()
	} else {
		s = app.ErrorDisplay(err)
	}
	l := widget.NewLabel(s)
	l.TextStyle.Monospace = true
	l.Wrapping = fyne.TextWrapBreak
	c := container.NewVScroll(container.NewBorder(title, nil, nil, nil, l))
	c.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", c, parent)
	xdesktop.DisableShortcutsForDialog(d, parent)
	d.Show()
}
