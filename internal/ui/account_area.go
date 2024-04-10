package ui

import (
	"context"
	"example/evebuddy/internal/logic"
	"example/evebuddy/internal/model"
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

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters *fyne.Container
	content    *fyne.Container
	ui         *ui
}

// func (u *ui) ShowManageDialog() {
// 	m := u.NewManageArea()
// 	m.Redraw()
// 	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
// 		m.showAddCharacterDialog()
// 	})
// 	button.Importance = widget.HighImportance
// 	c := container.NewScroll(m.content)
// 	c.SetMinSize(fyne.NewSize(400, 400))
// 	content := container.NewBorder(button, nil, nil, nil, c)
// 	dialog := dialog.NewCustom("Manage Characters", "Close", content, u.window)
// 	m.dialog = dialog
// 	dialog.SetOnClosed(func() {
// 		u.characterArea.Redraw()
// 	})
// 	dialog.Show()
// }

func (u *ui) NewAccountArea() *accountArea {
	m := &accountArea{
		ui: u,
	}
	characters := container.NewVBox()
	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		m.showAddCharacterDialog()
	})
	button.Importance = widget.HighImportance
	c := container.NewScroll(characters)
	content := container.NewBorder(nil, button, nil, nil, c)
	m.characters = characters
	m.content = content
	m.Redraw()
	return m
}

func (m *accountArea) Redraw() {
	chars, err := model.FetchAllCharacters()
	if err != nil {
		panic(err)
	}
	m.characters.RemoveAll()
	for _, char := range chars {
		badge := makeCharacterBadge(char)
		selectButton := widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
			m.ui.SetCurrentCharacter(&char)
			m.ui.folderArea.Redraw()
		})
		isCurrentChar := char.ID == m.ui.CurrentCharID()
		if isCurrentChar {
			selectButton.Disable()
		}
		deleteButton := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
			dialog := dialog.NewConfirm(
				"Delete Character",
				fmt.Sprintf("Are you sure you want to delete %s?", char.Name),
				func(confirmed bool) {
					if confirmed {
						err := char.Delete()
						if err != nil {
							d := dialog.NewError(err, m.ui.window)
							d.Show()
						}
						m.Redraw()
						if isCurrentChar {
							c, err := model.FetchFirstCharacter()
							if err != nil {
								m.ui.ResetCurrentCharacter()
								m.ui.folderArea.Redraw()
							} else {
								m.ui.SetCurrentCharacter(c)
							}
						}
					}
				},
				m.ui.window,
			)
			dialog.Show()
		})
		deleteButton.Importance = widget.DangerImportance
		item := container.NewHBox(badge, layout.NewSpacer(), selectButton, deleteButton)
		m.characters.Add(item)
		m.characters.Add(widget.NewSeparator())
	}
	m.characters.Refresh()
}

func makeCharacterBadge(char model.Character) *fyne.Container {
	var charName, corpName string
	var charURI, corpURI fyne.URI

	charName = char.Name
	corpName = char.Corporation.Name
	charURI = char.PortraitURL(defaultIconSize)
	corpURI = char.Corporation.ImageURL(defaultIconSize)

	charImage := canvas.NewImageFromURI(charURI)
	charImage.FillMode = canvas.ImageFillOriginal
	corpImage := canvas.NewImageFromURI(corpURI)
	corpImage.FillMode = canvas.ImageFillOriginal

	color := theme.ForegroundColor()
	character := canvas.NewText(charName, color)
	character.TextStyle = fyne.TextStyle{Bold: true}
	corporation := canvas.NewText(corpName, color)
	names := container.NewVBox(character, corporation)
	content := container.NewHBox(charImage, corpImage, names)
	return content
}

func (m *accountArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.Background())
	dialog := dialog.NewCustom(
		"Add Character",
		"Cancel",
		widget.NewLabel("Please follow instructions in your browser to add a new character."),
		m.ui.window,
	)
	dialog.SetOnClosed(cancel)
	go func() {
		defer cancel()
		defer dialog.Hide()
		_, err := logic.AddCharacter(ctx)
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
		} else {
			m.Redraw()
		}
	}()
	dialog.Show()
}
