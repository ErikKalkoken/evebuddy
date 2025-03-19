package widget

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"

	"github.com/ErikKalkoken/evebuddy/internal/humanize"
)

func ShowInformationDialog(title, message string, parent fyne.Window) {
	d := dialog.NewInformation(title, message, parent)
	kxdialog.AddDialogKeyHandler(d, parent)
	d.Show()
}

// ShowErrorDialog shows a new custom error dialog.
func ShowErrorDialog(message string, err error, parent fyne.Window) {
	text := widget.NewLabel(fmt.Sprintf("%s\n\n%s", message, humanize.Error(err)))
	text.Wrapping = fyne.TextWrapWord
	text.Importance = widget.DangerImportance
	x := container.NewVScroll(text)
	x.SetMinSize(fyne.Size{Width: 400, Height: 100})
	d := dialog.NewCustom("Error", "OK", x, parent)
	kxdialog.AddDialogKeyHandler(d, parent)
	d.Show()
}
