package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
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
	icon         *widget.Icon
	name         *widget.Label
	switchButton *widgets.ContextMenuButton
	ui           *ui
}

func (u *ui) newToolbarArea() *toolbarArea {
	a := &toolbarArea{ui: u}
	a.icon = widget.NewIcon(theme.AccountIcon())
	a.name = widget.NewLabel("")
	a.switchButton = widgets.NewContextMenuButtonWithIcon(
		theme.NewThemedResource(resourceSwitchaccountSvg), "", fyne.NewMenu(""))
	c := container.NewHBox(
		container.NewHBox(a.icon, a.name, a.switchButton),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			u.ShowAboutDialog()
		}),
		widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			u.ShowSettingsDialog()
		}),
		widget.NewButtonWithIcon("", theme.NewThemedResource(resourceManageaccountsSvg), func() {
			u.ShowAccountDialog()
		}),
	)
	a.content = container.NewVBox(c, widget.NewSeparator())

	return a
}

func (a *toolbarArea) Refresh() {
	c := a.ui.CurrentChar()
	if c == nil {
		a.icon.SetResource(theme.AccountIcon())
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
	} else {
		r := a.ui.imageManager.CharacterPortrait(c.ID, defaultIconSize)
		a.icon.SetResource(r)
		s := fmt.Sprintf("%s (%s)", c.Character.Name, c.Character.Corporation.Name)
		a.name.Text = s
		a.name.TextStyle = fyne.TextStyle{Bold: true}
	}
	a.name.Refresh()

	menuItems, err := a.makeMenuItems(c)
	if err != nil {
		msg := "Failed to create switch menu"
		slog.Error(msg, "err", err)
		a.ui.statusArea.SetError(msg)
		return
	}
	a.switchButton.SetMenuItems(menuItems)
}

func (a *toolbarArea) makeMenuItems(c *model.MyCharacter) ([]*fyne.MenuItem, error) {
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := a.ui.service.ListMyCharactersShort()
	if err != nil {
		return menuItems, err
	}
	for _, myC := range cc {
		if c != nil && myC.ID == c.ID {
			continue
		}
		i := fyne.NewMenuItem(myC.Name, func() {
			newChar, err := a.ui.service.GetMyCharacter(myC.ID)
			if err != nil {
				msg := "Failed to create switch menu"
				slog.Error(msg, "err", err)
				a.ui.statusArea.SetError(msg)
				return

			}
			a.ui.SetCurrentCharacter(newChar)
		})
		i.Icon = a.ui.imageManager.CharacterPortrait(myC.ID, defaultIconSize)
		menuItems = append(menuItems, i)
	}
	return menuItems, nil
}
