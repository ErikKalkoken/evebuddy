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

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/widgets"
)

// toolbarArea is the UI area showing the current status aka status bar.
type toolbarArea struct {
	content      *fyne.Container
	icon         *canvas.Image
	name         *widget.Label
	switchButton *widgets.ContextMenuButton
	manageButton *widget.Button
	ui           *ui
}

func (u *ui) newToolbarArea() *toolbarArea {
	a := &toolbarArea{ui: u}
	a.icon = canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
	a.icon.FillMode = canvas.ImageFillContain
	a.icon.SetMinSize(fyne.Size{Width: defaultIconSize, Height: defaultIconSize})
	a.name = widget.NewLabel("")
	a.switchButton = widgets.NewContextMenuButtonWithIcon(
		theme.NewThemedResource(resourceSwitchaccountSvg), "", fyne.NewMenu(""))
	a.manageButton = widget.NewButtonWithIcon("", theme.NewThemedResource(resourceManageaccountsSvg), func() {
		u.showAccountDialog()
	})
	c := container.NewHBox(
		container.NewHBox(a.icon, a.name, a.switchButton),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			u.showAboutDialog()
		}),
		widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			u.showSettingsDialog()
		}),
		a.manageButton,
	)
	a.content = container.NewVBox(c, widget.NewSeparator())
	return a
}

func (a *toolbarArea) refresh() {
	c := a.ui.currentChar()
	if c == nil {
		a.icon.Resource = resourceCharacterplaceholder32Jpeg
		a.icon.Refresh()
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
	} else {
		r, err := a.ui.sv.EveImage.CharacterPortrait(c.ID, defaultIconSize)
		if err != nil {
			slog.Error("Failed to fetch character portrait", "characterID", c.ID, "err", err)
			r = resourceCharacterplaceholder32Jpeg
		}
		a.icon.Resource = r
		a.icon.Refresh()
		s := fmt.Sprintf("%s (%s)", c.EveCharacter.Name, c.EveCharacter.Corporation.Name)
		a.name.Text = s
		a.name.TextStyle = fyne.TextStyle{Bold: true}
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
	if a.ui.currentChar() == nil {
		a.manageButton.Importance = widget.HighImportance
		a.manageButton.Refresh()
	} else {
		a.manageButton.Importance = widget.MediumImportance
		a.manageButton.Refresh()
	}

}

func (a *toolbarArea) makeMenuItems(c *model.Character) ([]*fyne.MenuItem, error) {
	ctx := context.Background()
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := a.ui.sv.Characters.ListCharactersShort(ctx)
	if err != nil {
		return menuItems, err
	}
	for _, myC := range cc {
		if c != nil && myC.ID == c.ID {
			continue
		}
		item := fyne.NewMenuItem(myC.Name, func() {
			err := a.ui.loadCurrentCharacter(ctx, myC.ID)
			if err != nil {
				msg := "Failed to switch to new character"
				slog.Error(msg, "err", err)
				a.ui.statusBarArea.SetError(msg)
				return

			}
		})
		item.Icon = resourceCharacterplaceholder32Jpeg
		go func() {
			r, err := a.ui.sv.EveImage.CharacterPortrait(myC.ID, defaultIconSize)
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
