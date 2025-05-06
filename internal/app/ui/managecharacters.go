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

	kmodal "github.com/ErikKalkoken/fyne-kx/modal"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type accountCharacter struct {
	id           int32
	name         string
	missingToken bool
}

type ManageCharacters struct {
	widget.BaseWidget

	OnSelectCharacter func()
	OnUpdate          func(characterCount int)

	characters   []accountCharacter
	list         *widget.List
	sb           *iwidget.Snackbar
	showSnackbar func(string)
	title        *widget.Label
	u            *BaseUI
	window       fyne.Window
}

func NewManageCharacters(u *BaseUI) *ManageCharacters {
	a := &ManageCharacters{
		characters:   make([]accountCharacter, 0),
		showSnackbar: u.ShowSnackbar,
		title:        appwidget.MakeTopLabel(),
		window:       u.MainWindow(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeCharacterList()
	return a
}

func (a *ManageCharacters) CreateRenderer() fyne.WidgetRenderer {
	var c fyne.CanvasObject
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.ShowAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.u.IsOffline() {
		add.Disable()
	}
	if a.u.isDesktop() {
		p := theme.Padding()
		c = container.NewBorder(
			a.title,
			container.NewCenter(container.New(layout.NewCustomPaddedLayout(p, p, 0, 0), add)),
			nil,
			nil,
			a.list,
		)
	} else {
		c = container.NewBorder(
			a.title,
			nil,
			nil,
			nil,
			a.list,
		)
	}
	return widget.NewSimpleRenderer(c)
}

func (a *ManageCharacters) SetWindow(w fyne.Window) {
	a.window = w
	if a.sb != nil {
		a.sb.Stop()
	}
	a.sb = iwidget.NewSnackbar(w)
	a.sb.Start()
	a.showSnackbar = func(s string) {
		a.sb.Show(s)
	}
}

func (a *ManageCharacters) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			portrait := iwidget.NewImageFromResource(
				icons.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			button := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
			button.Importance = widget.DangerImportance
			issue := widget.NewLabel("Scope issue - please re-add!")
			issue.Importance = widget.WarningImportance
			issue.Hide()
			row := container.NewHBox(portrait, name, issue, layout.NewSpacer(), button)
			return row
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
			go a.u.updateAvatar(c.id, func(r fyne.Resource) {
				fyne.Do(func() {
					icon.Resource = r
					icon.Refresh()
				})
			})

			issue := row[2].(*widget.Label)
			if c.missingToken {
				issue.Show()
			} else {
				issue.Hide()
			}

			row[4].(*widget.Button).OnTapped = func() {
				a.showDeleteDialog(c)
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.characters) {
			return
		}
		c := a.characters[id]
		if err := a.u.loadCharacter(c.id); err != nil {
			slog.Error("load current character", "char", c, "err", err)
			return
		}
		a.u.updateStatus()
		if a.OnSelectCharacter != nil {
			a.OnSelectCharacter()
		}
	}
	return l
}

func (a *ManageCharacters) showDeleteDialog(c accountCharacter) {
	a.u.ShowConfirmDialog(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s with all it's locally stored data?", c.name),
		"Delete",
		func(confirmed bool) {
			if confirmed {
				m := kmodal.NewProgressInfinite(
					"Deleting character",
					fmt.Sprintf("Deleting %s...", c.name),
					func() error {
						err := a.u.cs.DeleteCharacter(context.TODO(), c.id)
						if err != nil {
							return err
						}
						a.update()
						return nil
					},
					a.window,
				)
				m.OnSuccess = func() {
					a.showSnackbar(fmt.Sprintf("Character %s deleted", c.name))
					go func() {
						a.update()
						if a.u.currentCharacterID() == c.id {
							a.u.setAnyCharacter()
						}
						a.u.updateCrossPages()
						a.u.updateStatus()
					}()
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete character", "characterID", c.id)
					a.showSnackbar(fmt.Sprintf("ERROR: Failed to delete character %s", c.name))
				}
				m.Start()
			}
		},
		a.window,
	)
}

func (a *ManageCharacters) update() {
	characters := xslices.Map(a.u.scs.ListCharacters(), func(c *app.CharacterShort) accountCharacter {
		return accountCharacter{id: c.ID, name: c.Name}
	})
	// hasToken, err := a.u.cs.HasTokenWithScopes(context.Background(), c.ID)
	// if err != nil {
	// 	slog.Error("Tried to check if character has token", "err", err)
	// 	hasToken = true // do not report error when state is unclear
	// }
	fyne.Do(func() {
		a.characters = characters
		a.list.Refresh()
		a.title.SetText(fmt.Sprintf("Characters (%d)", len(characters)))
	})
	if a.OnUpdate != nil {
		a.OnUpdate(len(characters))
	}
}

func (a *ManageCharacters) ShowAddCharacterDialog() {
	cancelCTX, cancel := context.WithCancel(context.Background())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.window,
	)
	a.u.ModifyShortcutsForDialog(d1, a.window)
	d1.SetOnClosed(cancel)
	go func() {
		err := func() error {
			characterID, err := a.u.cs.UpdateOrCreateCharacterFromSSO(cancelCTX, infoText)
			if errors.Is(err, app.ErrAborted) {
				return nil
			} else if err != nil {
				return err
			}
			a.update()
			go func() {
				if !a.u.hasCharacter() {
					a.u.loadCharacter(characterID)
				}
				a.u.updateStatus()
				a.u.updateCrossPages()
				ctx := context.Background()
				a.u.updateCharacterAndRefreshIfNeeded(ctx, characterID, true)
			}()
			return nil
		}()
		fyne.Do(func() {
			d1.Hide()
			if err != nil {
				slog.Error("Failed to add a new character", "error", err)
				a.u.ShowErrorDialog("Failed add a new character", err, a.window)
			}
		})
	}()
	fyne.Do(func() {
		d1.Show()
	})
}
