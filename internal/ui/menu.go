package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func MakeMenu(a fyne.App, ui *ui) *fyne.MainMenu {
	file := fyne.NewMenu("File")

	manageItem := fyne.NewMenuItem("Manage", func() {
		showManageDialog(ui)
	})
	character := fyne.NewMenu("Character", manageItem)

	aboutItem := fyne.NewMenuItem("About", func() {
		d := dialog.NewInformation("About", "esiapp v0.1.0", ui.window)
		d.Show()
	})
	help := fyne.NewMenu("Help", aboutItem)

	main := fyne.NewMainMenu(file, character, help)
	return main
}
