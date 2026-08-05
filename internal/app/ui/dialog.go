package ui

import (
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xdesktop"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
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

// ShowProgressConfirm shows a custom confirmation dialog with progress indicator.
// Note that the callback is called on confirmation only and runs in a goroutine.
func ShowProgressConfirm(
	title, message, confirmLabel string,
	confirmImportance widget.Importance,
	callback func(),
	parent fyne.Window,
) {
	text := widget.NewLabel(message)
	text.Alignment = fyne.TextAlignCenter
	text.Wrapping = fyne.TextWrapWord
	var d dialog.Dialog
	dismiss := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		d.Hide()
	})
	confirm := xwidget.NewProgressButton(confirmLabel, theme.ConfirmIcon(), func() {
		callback()
		fyne.Do(func() {
			d.Hide()
		})
	})
	confirm.SetImportance(confirmImportance)
	p := theme.Padding()
	buttons := container.NewGridWithColumns(
		2,
		container.NewHBox(layout.NewSpacer(), container.NewCenter(dismiss)),
		container.NewHBox(container.NewCenter(confirm), layout.NewSpacer()),
	)
	minimumTextSize := canvas.NewRectangle(color.Transparent)
	s := buttons.MinSize()
	minimumTextSize.SetMinSize(fyne.NewSize(s.Width*2, s.Height))
	d = dialog.NewCustomWithoutButtons(
		title,
		container.NewBorder(
			nil,
			container.New(layout.NewCustomPaddedLayout(4*p, 0, 0, 0), buttons),
			nil,
			nil,
			container.NewStack(minimumTextSize, text),
		),
		parent,
	)
	xdesktop.DisableShortcutsForDialog(d, parent)
	d.Show()
}

// ShowErrorAndLog shows a error dialog and logs the error.
func ShowErrorAndLog(message string, err error, IsDeveloperMode bool, parent fyne.Window) {
	slog.Error(message, "error", err)
	var s string
	if IsDeveloperMode {
		s = err.Error()
	} else {
		s = app.ErrorDisplay(err)
	}
	errMessage := widget.NewLabel(s)
	errMessage.TextStyle.Monospace = true
	errMessage.Wrapping = fyne.TextWrapBreak
	errMessage.Importance = widget.DangerImportance
	c := container.NewVScroll(container.NewBorder(
		widget.NewLabel(message),
		nil,
		nil,
		nil,
		errMessage,
	))
	c.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", c, parent)
	xdesktop.DisableShortcutsForDialog(d, parent)
	d.Show()
}
