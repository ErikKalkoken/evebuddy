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

// manageArea is the UI area for managing of characters.
type manageArea struct {
	content        *fyne.Container
	dialog         *dialog.CustomDialog
	selectedCharID int32
	ui             *ui
}

func newManageArea(u *ui) *manageArea {
	content := container.NewVBox()
	m := &manageArea{
		ui:             u,
		content:        content,
		selectedCharID: u.characterArea.currentCharID,
	}
	return m
}

func (m *manageArea) update() {
	chars, err := model.FetchAllCharacters()
	if err != nil {
		panic(err)
	}
	m.content.RemoveAll()
	for _, char := range chars {
		uri := char.PortraitURL(defaultIconSize)
		image := canvas.NewImageFromURI(uri)
		image.FillMode = canvas.ImageFillOriginal
		name := widget.NewLabel(char.Name)
		selectButton := widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
			m.selectedCharID = char.ID
			m.dialog.Hide()
		})
		isCurrentChar := char.ID == m.ui.characterArea.currentCharID
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
						m.update()
						if isCurrentChar {
							m.ui.characterArea.update(0)
						}
					}
				},
				m.ui.window,
			)
			dialog.Show()
		})
		deleteButton.Importance = widget.DangerImportance
		item := container.NewHBox(image, name, layout.NewSpacer(), selectButton, deleteButton)
		m.content.Add(item)
		m.content.Add(widget.NewSeparator())
	}
	m.content.Refresh()
}

func (u *ui) showManageDialog() {
	m := newManageArea(u)
	m.update()
	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		showAddCharacterDialog(u.window, m)
	})
	button.Importance = widget.HighImportance
	c := container.NewScroll(m.content)
	c.SetMinSize(fyne.NewSize(400, 400))
	content := container.NewBorder(button, nil, nil, nil, c)
	dialog := dialog.NewCustom("Manage Characters", "Close", content, u.window)
	m.dialog = dialog
	dialog.SetOnClosed(func() {
		u.characterArea.update(m.selectedCharID)
	})
	dialog.Show()
}

func showAddCharacterDialog(w fyne.Window, m *manageArea) {
	ctx, cancel := context.WithCancel(context.Background())
	dialog := dialog.NewCustom(
		"Add Character",
		"Cancel",
		widget.NewLabel("Please follow instructions in your browser to add a new character."),
		w,
	)
	dialog.SetOnClosed(cancel)
	go func() {
		defer cancel()
		defer dialog.Hide()
		_, err := AddCharacter(ctx)
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
		} else {
			m.update()
		}
	}()
	dialog.Show()
}