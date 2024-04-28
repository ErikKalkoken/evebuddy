package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"example/evebuddy/internal/api/images"
	"example/evebuddy/internal/storage"
)

// accountArea is the UI area for managing of characters.
type accountArea struct {
	content *fyne.Container
	dialog  *dialog.CustomDialog
	ui      *ui
}

func (u *ui) ShowAccountDialog() {
	m := u.NewAccountArea()
	m.Redraw()
	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		m.showAddCharacterDialog()
	})
	button.Importance = widget.HighImportance
	c := container.NewScroll(m.content)
	c.SetMinSize(fyne.NewSize(400, 400))
	content := container.NewBorder(button, nil, nil, nil, c)
	dialog := dialog.NewCustom("Manage Characters", "Close", content, u.window)
	m.dialog = dialog
	dialog.SetOnClosed(func() {
		u.characterArea.Redraw()
	})
	dialog.Show()
}

func (u *ui) NewAccountArea() *accountArea {
	content := container.NewVBox()
	m := &accountArea{
		ui:      u,
		content: content,
	}
	return m
}

func (m *accountArea) Redraw() {
	chars, err := m.ui.service.ListMyCharacters()
	if err != nil {
		panic(err)
	}
	m.content.RemoveAll()
	for _, char := range chars {
		uri, _ := images.CharacterPortraitURL(char.ID, defaultIconSize)
		image := canvas.NewImageFromURI(uri)
		image.FillMode = canvas.ImageFillOriginal
		name := widget.NewLabel(char.Name)
		selectButton := widget.NewButtonWithIcon("Select", theme.ConfirmIcon(), func() {
			character, err := m.ui.service.GetMyCharacter(char.ID)
			if err != nil {
				panic(err)
			}
			m.ui.SetCurrentCharacter(&character)
			m.dialog.Hide()
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
						err := m.ui.service.DeleteMyCharacter(char.ID)
						if err != nil {
							d := dialog.NewError(err, m.ui.window)
							d.Show()
						}
						m.Redraw()
						if isCurrentChar {
							c, err := m.ui.service.GetAnyMyCharacter()
							if err != nil {
								if errors.Is(err, storage.ErrNotFound) {
									m.ui.ResetCurrentCharacter()
								} else {
									panic(err)
								}
							} else {
								m.ui.SetCurrentCharacter(&c)
							}
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

func (m *accountArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.Background())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	dlg := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		m.ui.window,
	)
	dlg.SetOnClosed(cancel)
	errChan := make(chan error)
	go func() {
		defer cancel()
		defer dlg.Hide()
		err := m.ui.service.UpdateOrCreateMyCharacterFromSSO(ctx, infoText)
		errChan <- err
	}()
	dlg.Show()
	err := <-errChan
	if err != nil {
		slog.Error("Failed to add a new character", "error", err)
		dlg := dialog.NewInformation(
			"Error",
			fmt.Sprintf("An error occurred when trying to add a new character:\n%s", err),
			m.ui.window,
		)
		dlg.Show()
	} else {
		m.Redraw()
	}
}
