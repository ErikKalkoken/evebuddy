package desktopui

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

	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (u *DesktopUI) showAccountDialog() {
	err := func() error {
		currentChars := set.New[int32]()
		cc, err := u.CharacterService.ListCharactersShort(context.Background())
		if err != nil {
			return err
		}
		for _, c := range cc {
			currentChars.Add(c.ID)
		}
		a := u.newAccountArea()
		d := dialog.NewCustom("Manage Characters", "Close", a.content, u.window)
		kxdialog.AddDialogKeyHandler(d, u.window)
		a.dialog = d
		d.SetOnClosed(func() {
			defer a.u.enableMenuShortcuts()
			incomingChars := set.New[int32]()
			for _, c := range a.characters {
				incomingChars.Add(c.id)
			}
			if currentChars.Equal(incomingChars) {
				return
			}
			if !incomingChars.Contains(a.u.characterID()) {
				if err := a.u.setAnyCharacter(); err != nil {
					slog.Error("Failed to set any character", "error", err)
				}
			}
			if currentChars.Difference(incomingChars).Size() == 0 {
				// no char has been deleted but still need to update some cross info
				a.u.toolbarArea.refresh()
				u.statusBarArea.refreshCharacterCount()
				return
			}
			a.u.refreshCrossPages()
		})
		u.disableMenuShortcuts()
		d.Show()
		d.Resize(fyne.Size{Width: 500, Height: 500})
		if err := a.refresh(); err != nil {
			d.Hide()
			return err
		}
		return nil
	}()
	if err != nil {
		d := NewErrorDialog("Failed to show account dialog", err, u.window)
		d.Show()
	}
}

type accountCharacter struct {
	id                int32
	name              string
	hasTokenWithScope bool
}

// accountArea is the UI area for managing of characters.
type accountArea struct {
	characters []accountCharacter
	content    *fyne.Container
	dialog     *dialog.CustomDialog
	list       *widget.List
	title      *widget.Label
	u          *DesktopUI
}

func (u *DesktopUI) newAccountArea() *accountArea {
	a := &accountArea{
		characters: make([]accountCharacter, 0),
		title:      widget.NewLabel(""),
		u:          u,
	}

	a.list = a.makeCharacterList()
	a.title.TextStyle.Bold = true
	add := widget.NewButtonWithIcon("Add Character", theme.ContentAddIcon(), func() {
		a.showAddCharacterDialog()
	})
	add.Importance = widget.HighImportance
	if a.u.IsOffline {
		add.Disable()
	}
	a.content = container.NewBorder(
		a.title,
		container.NewVBox(add, container.NewPadded()),
		nil,
		nil,
		a.list,
	)
	return a
}

func (a *accountArea) makeCharacterList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			portrait := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			portrait.FillMode = canvas.ImageFillContain
			portrait.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
			name := widget.NewLabel("Template")
			button := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {})
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
			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.u.EveImageService.CharacterPortrait(c.id, defaultIconSize)
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
		if err := a.u.loadCharacter(context.TODO(), c.id); err != nil {
			slog.Error("load current character", "char", c, "err", err)
			return
		}
		a.dialog.Hide()
	}
	return l
}

func (a *accountArea) showDeleteDialog(c accountCharacter) {
	d1 := NewConfirmDialog(
		"Delete Character",
		fmt.Sprintf("Are you sure you want to delete character %s?", c.name),
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
						if err := a.refresh(); err != nil {
							return err
						}
						return nil
					},
					a.u.window,
				)
				m.OnSuccess = func() {
					d := dialog.NewInformation("Delete Character", fmt.Sprintf("Character %s deleted", c.name), a.u.window)
					kxdialog.AddDialogKeyHandler(d, a.u.window)
					d.Show()
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete character", "characterID", c.id)
					d := NewErrorDialog(fmt.Sprintf("Failed to delete character %s", c.name), err, a.u.window)
					d.Show()
				}
				m.Start()
			}
		},
		a.u.window,
	)
	d1.Show()
}

func (a *accountArea) refresh() error {
	cc, err := a.u.CharacterService.ListCharactersShort(context.TODO())
	if err != nil {
		return err
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
	a.title.SetText(fmt.Sprintf("Characters (%d)", len(a.characters)))
	return nil
}

func (a *accountArea) showAddCharacterDialog() {
	cancelCTX, cancel := context.WithCancel(context.TODO())
	s := "Please follow instructions in your browser to add a new character."
	infoText := binding.BindString(&s)
	content := widget.NewLabelWithData(infoText)
	d1 := dialog.NewCustom(
		"Add Character",
		"Cancel",
		content,
		a.u.window,
	)
	kxdialog.AddDialogKeyHandler(d1, a.u.window)
	d1.SetOnClosed(cancel)
	go func() {
		err := func() error {
			characterID, err := a.u.CharacterService.UpdateOrCreateCharacterFromSSO(cancelCTX, infoText)
			if errors.Is(err, character.ErrAborted) {
				return nil
			} else if err != nil {
				return err
			}
			if err := a.refresh(); err != nil {
				return err
			}
			a.u.updateCharacterAndRefreshIfNeeded(context.Background(), characterID, false)
			return nil
		}()
		d1.Hide()
		if err != nil {
			slog.Error("Failed to add a new character", "error", err)
			d2 := NewErrorDialog("Failed add a new character", err, a.u.window)
			d2.Show()
		}
	}()
	d1.Show()
}
