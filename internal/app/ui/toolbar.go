package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
)

// toolbarArea is the UI area showing the current status aka status bar.
type toolbarArea struct {
	content      *fyne.Container
	icon         *canvas.Image
	name         *widget.Label
	switchButton *widgets.ContextMenuButton
	ui           *ui
}

func (u *ui) newToolbarArea() *toolbarArea {
	a := &toolbarArea{ui: u}
	a.icon = canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
	a.icon.FillMode = canvas.ImageFillContain
	a.icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
	a.name = widget.NewLabel("")
	a.switchButton = widgets.NewContextMenuButtonWithIcon(
		theme.NewThemedResource(resourceSwitchaccountSvg), "Switch", fyne.NewMenu(""))
	c := container.NewHBox(container.NewPadded(a.icon), a.name, layout.NewSpacer(), a.switchButton)
	a.content = container.NewVBox(c, widget.NewSeparator())
	return a
}

func (a *toolbarArea) refresh() {
	c := a.ui.currentCharacter()
	if c == nil {
		a.icon.Resource = resourceCharacterplaceholder32Jpeg
		a.icon.Refresh()
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
		a.name.Importance = widget.LowImportance
	} else {
		r, err := a.ui.EveImageService.CharacterPortrait(c.ID, defaultIconSize)
		if err != nil {
			slog.Error("Failed to fetch character portrait", "characterID", c.ID, "err", err)
			r = resourceCharacterplaceholder32Jpeg
		}
		a.icon.Resource = r
		a.icon.Refresh()
		s := fmt.Sprintf("%s (%s)", c.EveCharacter.Name, c.EveCharacter.Corporation.Name)
		a.name.Text = s
		a.name.TextStyle = fyne.TextStyle{Bold: true}
		a.name.Importance = widget.MediumImportance
	}
	a.name.Refresh()

	menuItems, err := a.makeMenuItems(c)
	if err != nil {
		msg := "Failed to create switch menu"
		slog.Error(msg, "err", err)
		a.ui.statusBarArea.SetError(msg)
		return
	}
	a.switchButton.SetMenuItems(menuItems)
	if len(menuItems) == 0 {
		a.switchButton.Disable()
	} else {
		a.switchButton.Enable()
	}
}

func (a *toolbarArea) makeMenuItems(c *app.Character) ([]*fyne.MenuItem, error) {
	ctx := context.TODO()
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := a.ui.CharacterService.ListCharactersShort(ctx)
	if err != nil {
		return menuItems, err
	}
	for _, myC := range cc {
		if c != nil && myC.ID == c.ID {
			continue
		}
		item := fyne.NewMenuItem(myC.Name, func() {
			err := a.ui.loadCharacter(ctx, myC.ID)
			if err != nil {
				msg := "Failed to switch to new character"
				slog.Error(msg, "err", err)
				a.ui.showErrorDialog(msg, err)
				return

			}
		})
		item.Icon = resourceCharacterplaceholder32Jpeg
		go func() {
			r, err := a.ui.EveImageService.CharacterPortrait(myC.ID, defaultIconSize)
			if err != nil {
				slog.Error("Failed to fetch character portrait", "characterID", myC.ID, "err", err)
				r = resourceCharacterplaceholder32Jpeg
			}
			item.Icon = r
			a.switchButton.Refresh()
		}()
		menuItems = append(menuItems, item)
	}
	return menuItems, nil
}
