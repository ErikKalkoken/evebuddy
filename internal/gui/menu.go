package gui

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func MakeMenu(a fyne.App, e *esiApp) *fyne.MainMenu {
	file := fyne.NewMenu("File")

	manageItem := fyne.NewMenuItem("Manage", func() {
		w2 := a.NewWindow("Manage Characters")
		b := widget.NewButton("Add Character", func() {
			ctx, cancel := context.WithCancel(context.Background())
			dlg := dialog.NewCustom(
				"Add Character",
				"Cancel",
				widget.NewLabel("Please follow instructions in your browser to add a new character."),
				e.Main,
			)
			dlg.SetOnClosed(cancel)
			go func() {
				defer cancel()
				defer dlg.Hide()
				token, err := AddCharacter(ctx)
				if err != nil {
					slog.Error("Failed to add a new character", "error", err)
				} else {
					e.characters.update(token.CharacterID)
				}
			}()
			dlg.Show()
		})
		c := container.NewBorder(nil, b, nil, nil)
		w2.SetContent(c)
		w2.Resize(fyne.NewSize(600, 400))
		w2.Show()
	})
	character := fyne.NewMenu("Character", manageItem)

	aboutItem := fyne.NewMenuItem("About", func() {
		d := dialog.NewInformation("About", "esiapp v0.1.0", e.Main)
		d.Show()
	})
	help := fyne.NewMenu("Help", aboutItem)

	main := fyne.NewMainMenu(file, character, help)
	return main
}
