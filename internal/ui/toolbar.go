package ui

import (
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
		u.ShowAccountDialog()
	})
	c := container.NewHBox(
		container.NewHBox(a.icon, a.name, a.switchButton),
		layout.NewSpacer(),
		widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			u.ShowAboutDialog()
		}),
		widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
			u.ShowSettingsDialog()
		}),
		a.manageButton,
	)
	a.content = container.NewVBox(c, widget.NewSeparator())

	return a
}

func (a *toolbarArea) Refresh() {
	c := a.ui.CurrentChar()
	if c == nil {
		a.icon.Resource = resourceCharacterplaceholder32Jpeg
		a.icon.Refresh()
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
	} else {
		r, err := a.ui.imageManager.CharacterPortrait(c.ID, defaultIconSize)
		if err != nil {
			panic(err)
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
		a.ui.statusArea.SetError(msg)
		return
	}
	a.switchButton.SetMenuItems(menuItems)
	if len(menuItems) == 0 {
		a.switchButton.Disable()
	} else {
		a.switchButton.Enable()
	}
	if a.ui.CurrentChar() == nil {
		a.manageButton.Importance = widget.HighImportance
		a.manageButton.Refresh()
	} else {
		a.manageButton.Importance = widget.MediumImportance
		a.manageButton.Refresh()
	}

}

func (a *toolbarArea) makeMenuItems(c *model.Character) ([]*fyne.MenuItem, error) {
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := a.ui.service.ListCharactersShort()
	if err != nil {
		return menuItems, err
	}
	for _, myC := range cc {
		if c != nil && myC.ID == c.ID {
			continue
		}
		i := fyne.NewMenuItem(myC.Name, func() {
			err := a.ui.LoadCurrentCharacter(myC.ID)
			if err != nil {
				msg := "Failed to switch to new character"
				slog.Error(msg, "err", err)
				a.ui.statusArea.SetError(msg)
				return

			}
		})
		r, err := a.ui.imageManager.CharacterPortrait(myC.ID, defaultIconSize)
		if err != nil {
			panic(err)
		}
		i.Icon = r
		menuItems = append(menuItems, i)
	}
	return menuItems, nil
}
