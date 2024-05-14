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

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
)

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters []*model.MyCharacterShort
	content    *fyne.Container
	dialog     *dialog.CustomDialog
	list       *widget.List
	total      *widget.Label
	ui         *ui
}

func (u *ui) ShowAccountDialog() {
	a := u.NewAccountArea()
	dialog := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
	a.dialog = dialog
	dialog.Show()
	dialog.Resize(fyne.Size{Width: 500, Height: 500})
	a.Refresh()
}

func (u *ui) NewAccountArea() *accountArea {
	a := &accountArea{
		characters: make([]*model.MyCharacterShort, 0),
		total:      widget.NewLabel(""),
		ui:         u,
	}

	a.list = widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			icon := widget.NewIcon(resourceCharacterplaceholder32Jpeg)
			name := widget.NewLabel("Template")
			b := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
			b.Importance = widget.DangerImportance
			row := container.NewHBox(icon, name, layout.NewSpacer(), b)
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
			c := a.characters[id]
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			r := u.imageManager.CharacterPortrait(c.ID, defaultIconSize)
			image := canvas.NewImageFromResource(r)
			icon.SetResource(image.Resource)
			row.Objects[1].(*widget.Label).SetText(c.Name)
			row.Objects[3].(*widget.Button).OnTapped = func() {
				d1 := dialog.NewConfirm(
					"Delete Character",
					fmt.Sprintf("Are you sure you want to delete %s?", c.Name),
					func(confirmed bool) {
						if confirmed {
							err := a.ui.service.DeleteMyCharacter(c.ID)
							if err != nil {
								d2 := dialog.NewError(err, a.ui.window)
								d2.Show()
							}
							a.Refresh()
							isCurrentChar := c.ID == a.ui.CurrentCharID()
							if isCurrentChar {
								err := a.ui.SetAnyCharacter()
								if err != nil {
									panic(err)
								}
							}
							u.RefreshOverview()
							a.ui.toolbarArea.Refresh()
						}
					},
					a.ui.window,
				)
				d1.Show()
			}
		})

	a.list.OnSelected = func(id widget.ListItemID) {
		a.list.UnselectAll() // Hack. Should be replaced by custom list widget.
	}

	button := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	button.Importance = widget.HighImportance
	a.content = container.NewBorder(button, a.total, nil, nil, container.NewScroll(a.list))
	return a
}

func (a *accountArea) Refresh() {
	a.characters = a.characters[0:0]
	var err error
	a.characters, err = a.ui.service.ListMyCharactersShort()
	if err != nil {
		panic(err)
	}
	a.list.Refresh()
	a.total.SetText(fmt.Sprintf("Characters: %d", len(a.characters)))
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
		err := a.ui.service.UpdateOrCreateMyCharacterFromSSO(ctx, infoText)
		if err != nil {
			if !errors.Is(err, service.ErrAborted) {
				slog.Error("Failed to add a new character", "error", err)
				d2 := dialog.NewInformation(
					"Error",
					fmt.Sprintf("An error occurred when trying to add a new character:\n%s", err),
					a.ui.window,
				)
				d2.Show()
			}
		} else {
			isFirst := len(a.characters) == 0
			a.Refresh()
			a.ui.RefreshOverview()
			a.ui.toolbarArea.Refresh()
			if isFirst {
				err := a.ui.SetAnyCharacter()
				if err != nil {
					panic(err)
				}
			}
		}
		d1.Hide()
	}()
	d1.Show()
}
