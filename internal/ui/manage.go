package ui

import (
	"context"
	"example/esiapp/internal/model"
	"fmt"
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
	content        *fyne.Container
	window         fyne.Window
	characters     *characterArea
	dialog         *dialog.CustomDialog
	selectedCharID int32
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
			c.selectedCharID = char.ID
			c.dialog.Hide()
		})
		isCurrentChar := char.ID == c.characters.currentCharID
		if isCurrentChar {
			btnSelect.Disable()
		}
		btnDelete := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			dlg := dialog.NewConfirm(
				"Delete Character",
				fmt.Sprintf("Are you sure you want to delete %s?", char.Name),
				func(confirmed bool) {
					if confirmed {
						err := char.Delete()
						if err != nil {
							d := dialog.NewError(err, c.window)
							d.Show()
						}
						c.update()
						if isCurrentChar {
							c.characters.update(0)
						}
					}
				},
				c.window,
			)
			dlg.Show()
		})
		btnDelete.Importance = widget.DangerImportance
		item := container.NewHBox(image, name, layout.NewSpacer(), btnSelect, btnDelete)
		c.content.Add(item)
		c.content.Add(widget.NewSeparator())
	}
	c.content.Refresh()
}

func newCharacterList(w fyne.Window, characters *characterArea) *characterList {
	content := container.NewVBox()
	c := &characterList{
		window:         w,
		content:        content,
		characters:     characters,
		selectedCharID: characters.currentCharID,
	}
	return c
}

func showManageDialog(e *eveApp) {
	c := newCharacterList(e.winMain, e.characterArea)
	c.update()
	btnAdd := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		showAddCharacterDialog(e.winMain, c)
	})
	btnAdd.Importance = widget.HighImportance
	c2 := container.NewScroll(c.content)
	c2.SetMinSize(fyne.NewSize(400, 400))
	content := container.NewBorder(btnAdd, nil, nil, nil, c2)
	dlg := dialog.NewCustom("Manage Characters", "Close", content, e.winMain)
	c.dialog = dlg
	dlg.SetOnClosed(func() {
		c.characters.update(c.selectedCharID)
	})
	dlg.Show()
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
