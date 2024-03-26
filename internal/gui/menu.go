package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func MakeMenu(a fyne.App, e *eveApp) *fyne.MainMenu {
	file := fyne.NewMenu("File")

	var w2 fyne.Window
	manageItem := fyne.NewMenuItem("Manage", func() {
		if w2 != nil {
			w2.Show()
		} else {
			w2 = makeManageWindow(a, e)
			w2.Show()
		}
	})
	character := fyne.NewMenu("Character", manageItem)

	aboutItem := fyne.NewMenuItem("About", func() {
		d := dialog.NewInformation("About", "esiapp v0.1.0", e.winMain)
		d.Show()
	})
	help := fyne.NewMenu("Help", aboutItem)

	main := fyne.NewMainMenu(file, character, help)
	return main
}
