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

	"github.com/ErikKalkoken/evebuddy/internal/service"
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
	total      *widget.Label
	ui         *ui
}

func (u *ui) ShowAccountDialog() {
	a := u.NewAccountArea()
	dialog := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
	a.dialog = dialog
	dialog.Show()
	dialog.Resize(fyne.Size{Width: 500, Height: 500})
	err := a.Refresh()
	if err != nil {
		u.statusArea.SetError("Failed to open dialog to manage characters")
		dialog.Hide()
	}
}

func (u *ui) NewAccountArea() *accountArea {
	a := &accountArea{
		characters: binding.NewUntypedList(),
		total:      widget.NewLabel(""),
		ui:         u,
	}

	list := widget.NewListWithData(
		a.characters,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
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
			icon := row.Objects[0].(*canvas.Image)
			r := u.imageManager.CharacterPortrait(c.id, defaultIconSize)
			image := canvas.NewImageFromResource(r)
			icon.Resource = image.Resource
			image.Refresh()
			name.SetText(c.name)
			row.Objects[3].(*widget.Button).OnTapped = func() {
				d1 := dialog.NewConfirm(
					"Delete Character",
					fmt.Sprintf("Are you sure you want to delete %s?", c.name),
					func(confirmed bool) {
						if confirmed {
							err := a.ui.service.DeleteMyCharacter(c.id)
							if err != nil {
								d2 := dialog.NewError(err, a.ui.window)
								d2.Show()
							}
							if err := a.Refresh(); err != nil {
								panic(err)
							}
							isCurrentChar := c.id == a.ui.CurrentCharID()
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

	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll() // Hack. Maybe replace with custom list widget?
	}

	b := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	b.Importance = widget.HighImportance
	a.content = container.NewBorder(b, a.total, nil, nil, container.NewScroll(list))
	return a
}

func (a *accountArea) Refresh() error {
	cc, err := a.ui.service.ListMyCharactersShort()
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
	a.total.SetText(fmt.Sprintf("Characters: %d", a.characters.Length()))
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
		characterID, err := a.ui.service.UpdateOrCreateMyCharacterFromSSO(ctx, infoText)
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
			isFirst := a.characters.Length() == 0
			if err := a.Refresh(); err != nil {
				panic(err)
			}
			a.ui.RefreshOverview()
			a.ui.toolbarArea.Refresh()
			if isFirst {
				if err := a.ui.SetAnyCharacter(); err != nil {
					panic(err)
				}
			} else {
				a.ui.overviewArea.MaybeUpdateAndRefresh(characterID)
			}
		}
		d1.Hide()
	}()
	d1.Show()
}
