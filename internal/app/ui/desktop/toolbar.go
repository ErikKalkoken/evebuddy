package desktop

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// toolbarArea is the UI area showing the current status aka status bar.
type toolbarArea struct {
	content      *fyne.Container
	icon         *canvas.Image
	name         *widget.Label
	switchButton *iwidget.ContextMenuButton
	u            *DesktopUI
}

func (u *DesktopUI) newToolbarArea() *toolbarArea {
	a := &toolbarArea{
		icon: iwidget.NewImageFromResource(
			ui.IconCharacterplaceholder64Jpeg,
			fyne.NewSquareSize(ui.DefaultIconUnitSize),
		),
		name: widget.NewLabel(""),
		switchButton: iwidget.NewContextMenuButtonWithIcon(
			theme.NewThemedResource(ui.IconSwitchaccountSvg), "Switch", fyne.NewMenu(""),
		),
		u: u,
	}
	c := container.NewHBox(container.NewPadded(a.icon), a.name, layout.NewSpacer(), a.switchButton)
	a.content = container.NewVBox(c, widget.NewSeparator())
	return a
}

func (a *toolbarArea) refresh() {
	c := a.u.CurrentCharacter()
	if c == nil {
		a.icon.Resource = ui.IconCharacterplaceholder64Jpeg
		a.icon.Refresh()
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
		a.name.Importance = widget.LowImportance
	} else {
		go a.u.UpdateAvatar(c.ID, func(r fyne.Resource) {
			a.icon.Resource = r
			a.icon.Refresh()
		})
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
		a.u.statusBarArea.SetError(msg)
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
	menuItems := make([]*fyne.MenuItem, 0)
	cc, err := a.u.CharacterService.ListCharactersShort(context.Background())
	if err != nil {
		return menuItems, err
	}
	for _, myC := range cc {
		if c != nil && myC.ID == c.ID {
			continue
		}
		item := fyne.NewMenuItem(myC.Name, func() {
			err := a.u.LoadCharacter(myC.ID)
			if err != nil {
				msg := "Failed to switch to new character"
				slog.Error(msg, "err", err)
				d := ui.NewErrorDialog(msg, err, a.u.Window)
				d.Show()
				return
			}
		})
		item.Icon = ui.IconCharacterplaceholder64Jpeg
		go a.u.UpdateAvatar(myC.ID, func(r fyne.Resource) {
			item.Icon = r
			a.switchButton.Refresh()
		})
		menuItems = append(menuItems, item)
	}
	return menuItems, nil
}
