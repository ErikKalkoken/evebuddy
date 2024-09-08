package uninstall

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/appdirs"
)

// RunApp runs the uninstall app
func RunApp(fyneApp fyne.App, ad appdirs.AppDirs) {
	w := fyneApp.NewWindow("Uninstall - EVE Buddy")
	label := widget.NewLabel(
		"Are you sure you want to uninstall this app\n" +
			"and delete all user files?")
	ok := widget.NewButtonWithIcon("OK", theme.ConfirmIcon(), func() {
		if err := ad.DeleteAll(); err != nil {
			closeApp(w, fmt.Sprintf("ERROR: %s", err))
			return
		}
		closeApp(w, "Files deleted")
	})
	cancel := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() {
		closeApp(w, "Aborted")
	})
	cancel.Importance = widget.HighImportance
	c := container.NewBorder(
		nil,
		container.NewHBox(cancel, layout.NewSpacer(), ok),
		nil,
		nil,
		container.NewCenter(label),
	)
	w.SetContent(c)
	w.Resize(fyne.Size{Width: 300, Height: 200})
	w.ShowAndRun()
}

func closeApp(w fyne.Window, message string) {
	d := dialog.NewInformation("Uninstall", message, w)
	d.SetOnClosed(w.Close)
	d.Show()
}
