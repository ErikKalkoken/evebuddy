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
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
)

type accountCharacter struct {
	id   int32
	name string
}

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters []accountCharacter
	content    *fyne.Container
	dialog     *dialog.CustomDialog
	bottom     *widget.Label
	ui         *ui
	list       *widget.List
}

func (u *ui) showAccountDialog() {
	a := u.newAccountArea()
	dialog := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
	a.dialog = dialog
	dialog.Show()
	dialog.Resize(fyne.Size{Width: 500, Height: 500})
	if err := a.Refresh(); err != nil {
		dialog.Hide()
		t := "Failed to open dialog to manage characters"
		slog.Error(t, "err", err)
		u.showErrorDialog(t, err)
	}
}

func (u *ui) newAccountArea() *accountArea {
	a := &accountArea{
		characters: make([]accountCharacter, 0),
		ui:         u,
	}

	a.bottom = widget.NewLabel("Hint: Click any character to enable it")
	a.bottom.Importance = widget.LowImportance
	a.bottom.Hide()

	a.list = a.makeCharacterList()

	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	b.Importance = widget.HighImportance
	a.content = container.NewBorder(b, a.bottom, nil, nil, a.list)
	return a
}

func (a *accountArea) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
			name := widget.NewLabel("Template")
			button := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
			button.Importance = widget.DangerImportance
			row := container.NewHBox(icon, name, layout.NewSpacer(), button)
			return row

			// hasToken, err := a.ui.service.HasTokenWithScopes(char.ID)
			// if err != nil {
			// 	slog.Error("Can not check if character has token", "err", err)
			// 	continue
			// }
			// if !hasToken {
			// 	row.Add(widget.NewIcon(theme.WarningIcon()))
			// }

		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.characters) {
				return
			}
			c := a.characters[id]
			row := co.(*fyne.Container).Objects
			name := row[1].(*widget.Label)
			name.SetText(c.name)

			icon := row[0].(*canvas.Image)
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.EveImageService.CharacterPortrait(c.id, defaultIconSize)
			})

			row[3].(*widget.Button).OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.characters) {
			return
		}
		c := a.characters[id]
		if err := a.ui.loadCharacter(context.TODO(), c.id); err != nil {
			slog.Error("failed to load current character", "char", c, "err", err)
			return
		}
		a.dialog.Hide()
	}
	return l
}

func (a *accountArea) showDeleteDialog(c accountCharacter) {
	d1 := dialog.NewConfirm(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s?", c.name),
		func(confirmed bool) {
			if confirmed {
				err := func(characterID int32) error {
					err := a.ui.CharacterService.DeleteCharacter(context.TODO(), characterID)
					if err != nil {
						return err
					}
					if err := a.Refresh(); err != nil {
						return err
					}
					isCurrentChar := characterID == a.ui.characterID()
					if isCurrentChar {
						err := a.ui.setAnyCharacter()
						if err != nil {
							return err
						}
					}
					a.ui.refreshOverview()
					a.ui.toolbarArea.refresh()
					return nil
				}(c.id)
				if err != nil {
					slog.Error("Failed to delete a character", "character", c, "err", err)
					a.ui.showErrorDialog("Failed to delete a character", err)
				}
			}
		},
		a.ui.window,
	)
	d1.Show()
}

func (a *accountArea) Refresh() error {
	cc, err := a.ui.CharacterService.ListCharactersShort(context.TODO())
	if err != nil {
		return err
	}
	cc2 := make([]accountCharacter, len(cc))
	for i, c := range cc {
		cc2[i] = accountCharacter{id: c.ID, name: c.Name}
	}
	a.characters = cc2
	a.list.Refresh()
	if len(cc2) > 0 {
		a.bottom.Show()
	} else {
		a.bottom.Hide()
	}
	return nil
}

func (a *accountArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.TODO())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.ui.window,
	)
	d1.SetOnClosed(cancel)
	go func() {
		err := func() error {
			defer cancel()
			characterID, err := a.ui.CharacterService.UpdateOrCreateCharacterFromSSO(ctx, infoText)
			if errors.Is(err, character.ErrAborted) {
				return nil
			} else if err != nil {
				return err
			}
			isFirst := len(a.characters) == 0
			if err := a.Refresh(); err != nil {
				return err
			}
			a.ui.refreshOverview()
			a.ui.toolbarArea.refresh()
			if isFirst {
				if err := a.ui.setAnyCharacter(); err != nil {
					return err
				}
			} else {
				a.ui.updateCharacterAndRefreshIfNeeded(ctx, characterID, false)
			}
			return nil
		}()
		d1.Hide()
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
			a.ui.showErrorDialog("Failed add a new character", err)
		}
	}()
	d1.Show()
}
