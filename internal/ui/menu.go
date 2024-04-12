package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func MakeMenu(a fyne.App, u *ui) *fyne.MainMenu {
	file := fyne.NewMenu("File")

	manageItem := fyne.NewMenuItem("Manage", func() {
		u.ShowManageDialog()
	})
	character := fyne.NewMenu("Character", manageItem)

	aboutItem := fyne.NewMenuItem("About", func() {
		text := "Eve Buddy v0.1.0\n\n(c) 2024 Erik Kalkoken"
		d := dialog.NewInformation("About", text, u.window)
		d.Show()
	})
	help := fyne.NewMenu("Help", aboutItem)

	main := fyne.NewMainMenu(file, character, help)
	return main
}
