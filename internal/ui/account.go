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
	"github.com/ErikKalkoken/evebuddy/internal/service/characters"
)

type accountCharacter struct {
	id   int32
	name string
}

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters binding.UntypedList
	content    *fyne.Container
	dialog     *dialog.CustomDialog
	bottom     *widget.Label
	ui         *ui
}

func (u *ui) showAccountDialog() {
	a := u.newAccountArea()
	dialog := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
	a.dialog = dialog
	dialog.Show()
	dialog.Resize(fyne.Size{Width: 500, Height: 500})
	err := a.Refresh()
	if err != nil {
		u.statusBarArea.SetError("Failed to open dialog to manage characters")
		dialog.Hide()
	}
}

func (u *ui) newAccountArea() *accountArea {
	a := &accountArea{
		characters: binding.NewUntypedList(),
		ui:         u,
	}

	a.bottom = widget.NewLabel("Hint: Click any character to enable it")
	a.bottom.Importance = widget.LowImportance
	a.bottom.Hide()

	list := a.makeCharacterList()

	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	b.Importance = widget.HighImportance
	a.content = container.NewBorder(b, a.bottom, nil, nil, list)
	return a
}

func (a *accountArea) makeCharacterList() *widget.List {
	l := widget.NewListWithData(
		a.characters,
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
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)

			name := row.Objects[1].(*widget.Label)
			c, err := convertDataItem[accountCharacter](di)
			if err != nil {
				slog.Error("failed to render row account table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(c.name)

			icon := row.Objects[0].(*canvas.Image)
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.imageManager.CharacterPortrait(c.id, defaultIconSize)
			})

			row.Objects[3].(*widget.Button).OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		c, err := getItemUntypedList[accountCharacter](a.characters, id)
		if err != nil {
			slog.Error("failed to access account character in list", "err", err)
			return
		}
		if err := a.ui.loadCurrentCharacter(context.Background(), c.id); err != nil {
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
					err := a.ui.service.Characters.DeleteCharacter(context.Background(), characterID)
					if err != nil {
						return err
					}
					if err := a.Refresh(); err != nil {
						return err
					}
					isCurrentChar := characterID == a.ui.currentCharID()
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
	cc, err := a.ui.service.Characters.ListCharactersShort(context.Background())
	if err != nil {
		return err
	}
	cc2 := make([]accountCharacter, len(cc))
	for i, c := range cc {
		cc2[i] = accountCharacter{id: c.ID, name: c.Name}
	}
	if err := a.characters.Set(copyToUntypedSlice(cc2)); err != nil {
		return err
	}
	if a.characters.Length() > 0 {
		a.bottom.Show()
	} else {
		a.bottom.Hide()
	}
	return nil
}

func (a *accountArea) showAddCharacterDialog() {
	ctx, cancel := context.WithCancel(context.Background())
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
			characterID, err := a.ui.service.Characters.UpdateOrCreateCharacterFromSSO(ctx, infoText)
			if errors.Is(err, characters.ErrAborted) {
				return nil
			} else if err != nil {
				return err
			}
			isFirst := a.characters.Length() == 0
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
