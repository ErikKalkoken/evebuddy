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
	content *fyne.Container
	window  fyne.Window
}

func (c *characterList) update() {
	chars, err := model.FetchAllCharacters()
	if err != nil {
		panic(err)
	}
	c.content.RemoveAll()
	for _, char := range chars {
		uri := char.PortraitURL(defaultIconSize)
		image := canvas.NewImageFromURI(uri)
		image.FillMode = canvas.ImageFillOriginal
		name := widget.NewLabel(char.Name)
		button := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			err := char.Delete()
			if err != nil {
				dlg := dialog.NewError(err, c.window)
				dlg.Show()
			}
			c.update()
		})
		button.Importance = widget.DangerImportance
		item := container.NewHBox(image, name, layout.NewSpacer(), button)
		c.content.Add(item)
		c.content.Add(widget.NewSeparator())
	}
	c.content.Refresh()
}

func newCharacterList(w fyne.Window) *characterList {
	content := container.NewVBox()
	c := &characterList{window: w, content: content}
	return c
}

func makeManageWindow(a fyne.App, e *eveApp) fyne.Window {
	w := a.NewWindow("Manage Characters")
	c := newCharacterList(w)
	c.update()
	btnAdd := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		showAddCharacterDialog(e.winMain, e.characters, c)
	})
	btnAdd.Importance = widget.HighImportance
	btnClose := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		w.Hide()
	})
	content := container.NewBorder(btnAdd, btnClose, nil, nil, c.content)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	return w
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
		_, err := AddCharacter(ctx)
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
		} else {
			// characters.update(token.CharacterID)
			list.update()
		}
	}()
	dlg.Show()
}
