package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func MakeMenu(w fyne.Window) *fyne.MainMenu {
	aboutItem := fyne.NewMenuItem("About", func() {
		d := dialog.NewInformation("About", "esiapp v0.1.0", w)
		d.Show()
	})
	file := fyne.NewMenu("File")
	help := fyne.NewMenu("Help", aboutItem)
	main := fyne.NewMainMenu(file, help)
	return main
}
