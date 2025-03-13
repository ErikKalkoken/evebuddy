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

	kxdialog "github.com/ErikKalkoken/fyne-kx/dialog"
	kmodal "github.com/ErikKalkoken/fyne-kx/modal"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type accountCharacter struct {
	id                int32
	name              string
	hasTokenWithScope bool
}

// AccountArea is the UI area for managing of characters.
type AccountArea struct {
	Content           fyne.CanvasObject
	OnSelectCharacter func()
	OnRefresh         func(characterCount int)

	snackbar   *iwidget.Snackbar
	characters []accountCharacter
	list       *widget.List
	title      *widget.Label
	window     fyne.Window
	u          *BaseUI
}

func NewAccountArea(u *BaseUI) *AccountArea {
	a := &AccountArea{
		characters: make([]accountCharacter, 0),
		snackbar:   u.Snackbar,
		title:      makeTopLabel(),
		window:     u.Window,
		u:          u,
	}

	a.list = a.makeCharacterList()
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.ShowAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.u.IsOffline {
		add.Disable()
	}
	close := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		a.window.Close()
	})
	if a.u.IsDesktop() {
		a.Content = container.NewBorder(
			a.title,
			container.NewHBox(layout.NewSpacer(), close, add, layout.NewSpacer()),
			nil,
			nil,
			a.list,
		)
	} else {
		a.Content = container.NewBorder(
			a.title,
			nil,
			nil,
			nil,
			a.list,
		)
	}
	return a
}

func (a *AccountArea) SetWindow(w fyne.Window) {
	a.window = w
	a.snackbar = iwidget.NewSnackbar(w)
	a.snackbar.Start()
}

func (a *AccountArea) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			portrait := iwidget.NewImageFromResource(
				icon.Characterplaceholder64Jpeg,
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
			go a.u.UpdateAvatar(c.id, func(r fyne.Resource) {
				icon.Resource = r
				icon.Refresh()
			})

			issue := row[2].(*widget.Label)
			if !c.hasTokenWithScope {
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
		if err := a.u.LoadCharacter(c.id); err != nil {
			slog.Error("load current character", "char", c, "err", err)
			return
		}
		a.u.RefreshStatus()
		if a.OnSelectCharacter != nil {
			a.OnSelectCharacter()
		}
	}
	return l
}

func (a *AccountArea) showDeleteDialog(c accountCharacter) {
	d1 := iwidget.NewConfirmDialog(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete %s with all it's locally stored data?", c.name),
		"Delete",
		func(confirmed bool) {
			if confirmed {
				m := kmodal.NewProgressInfinite(
					"Deleting character",
					fmt.Sprintf("Deleting %s...", c.name),
					func() error {
						err := a.u.CharacterService.DeleteCharacter(context.TODO(), c.id)
						if err != nil {
							return err
						}
						a.Refresh()
						return nil
					},
					a.window,
				)
				m.OnSuccess = func() {
					a.snackbar.Show(fmt.Sprintf("Character %s deleted", c.name))
					if a.u.CharacterID() == c.id {
						a.u.SetAnyCharacter()
					}
					a.u.RefreshCrossPages()
					a.u.RefreshStatus()
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete character", "characterID", c.id)
					a.snackbar.Show(fmt.Sprintf("ERROR: Failed to delete character %s", c.name))
				}
				m.Start()
			}
		},
		a.window,
	)
	d1.Show()
}

func (a *AccountArea) Refresh() {
	cc, err := a.u.CharacterService.ListCharactersShort(context.TODO())
	if err != nil {
		slog.Error("account refresh", "error", err)
		return
	}
	cc2 := make([]accountCharacter, len(cc))
	for i, c := range cc {
		hasToken, err := a.u.CharacterService.CharacterHasTokenWithScopes(context.Background(), c.ID)
		if err != nil {
			slog.Error("Tried to check if character has token", "err", err)
			hasToken = true // do not report error when state is unclear
		}
		cc2[i] = accountCharacter{id: c.ID, name: c.Name, hasTokenWithScope: hasToken}
	}
	a.characters = cc2
	a.list.Refresh()
	characterCount := len(a.characters)
	a.title.SetText(fmt.Sprintf("Characters (%d)", characterCount))
	if a.OnRefresh != nil {
		a.OnRefresh(characterCount)
	}
}

func (a *AccountArea) ShowAddCharacterDialog() {
	cancelCTX, cancel := context.WithCancel(context.TODO())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.window,
	)
	kxdialog.AddDialogKeyHandler(d1, a.window)
	d1.SetOnClosed(cancel)
	go func() {
		err := func() error {
			characterID, err := a.u.CharacterService.UpdateOrCreateCharacterFromSSO(cancelCTX, infoText)
			if errors.Is(err, character.ErrAborted) {
				return nil
			} else if err != nil {
				return err
			}
			a.Refresh()
			go a.u.UpdateCharacterAndRefreshIfNeeded(context.Background(), characterID, false)
			if !a.u.HasCharacter() {
				a.u.LoadCharacter(characterID)
			}
			a.u.RefreshCrossPages()
			a.u.RefreshStatus()
			return nil
		}()
		d1.Hide()
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
			d2 := iwidget.NewErrorDialog("Failed add a new character", err, a.window)
			d2.Show()
		}
	}()
	d1.Show()
}
