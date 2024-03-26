package gui

import (
	"context"
	"example/esiapp/internal/model"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type characterList struct {
	content fyne.CanvasObject
}

func (c *characterList) update() {
	chars, err := model.FetchAllCharacters()
	if err != nil {
		panic(err)
	}
	items := container.NewVBox()
	for _, c := range chars {
		uri := c.PortraitURL(defaultIconSize)
		image := canvas.NewImageFromURI(uri)
		image.FillMode = canvas.ImageFillOriginal

		label := widget.NewLabel(c.Name)
		button := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
		button.Importance = widget.DangerImportance
		i := container.NewHBox(image, label, layout.NewSpacer(), button)
		items.Add(i)
	}
	l := container.NewVScroll(items)
	c.content = l
	c.content.Refresh()
}

func makeManageWindow(a fyne.App, e *eveApp) fyne.Window {
	list := makeCharacterList()
	list.update()
	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		showAddCharacterDialog(e.winMain, e.characters, list)
	})
	content := container.NewBorder(nil, b, nil, nil, list.content)
	w := a.NewWindow("Manage Characters")
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	return w
}

func makeCharacterList() *characterList {
	return &characterList{}
}

func showAddCharacterDialog(w fyne.Window, characters *characters, list *characterList) {
	ctx, cancel := context.WithCancel(context.Background())
	dlg := dialog.NewCustom(
		"Add Character",
		"Cancel",
		widget.NewLabel("Please follow instructions in your browser to add a new character."),
		w,
	)
	dlg.SetOnClosed(cancel)
	go func() {
		defer cancel()
		defer dlg.Hide()
		token, err := AddCharacter(ctx)
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
		} else {
			characters.update(token.CharacterID)
			list.update()
		}
	}()
	dlg.Show()
}
