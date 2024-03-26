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
	content    *fyne.Container
	window     fyne.Window
	characters *characters
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
		btnSelect := widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
			c.characters.update(char.ID)
			c.window.Hide()
		})
		btnDelete := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			err := char.Delete()
			if err != nil {
				dlg := dialog.NewError(err, c.window)
				dlg.Show()
			}
			c.update()
			if char.ID == c.characters.currentCharID {
				c.characters.update(0)
			}
		})
		btnDelete.Importance = widget.DangerImportance
		item := container.NewHBox(image, name, layout.NewSpacer(), btnSelect, btnDelete)
		c.content.Add(item)
		c.content.Add(widget.NewSeparator())
	}
	c.content.Refresh()
}

func newCharacterList(w fyne.Window, characters *characters) *characterList {
	content := container.NewVBox()
	c := &characterList{window: w, content: content, characters: characters}
	return c
}

func makeManageWindow(a fyne.App, e *eveApp) fyne.Window {
	w := a.NewWindow("Manage Characters")
	c := newCharacterList(w, e.characters)
	c.update()
	btnAdd := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		showAddCharacterDialog(e.winMain, c)
	})
	btnAdd.Importance = widget.HighImportance
	btnClose := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		w.Hide()
	})
	content := container.NewBorder(btnAdd, btnClose, nil, nil, c.content)
	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.SetOnClosed(func() {
		e.winMain.RequestFocus()
	})
	return w
}

func showAddCharacterDialog(w fyne.Window, list *characterList) {
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
			list.update()
		}
	}()
	dlg.Show()
}
