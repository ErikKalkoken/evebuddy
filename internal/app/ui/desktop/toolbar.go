package desktop

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// toolbarArea is the UI area showing the current status aka status bar.
type toolbarArea struct {
	content *fyne.Container
	icon    *kwidget.TappableImage
	name    *widget.Label
	u       *DesktopUI
}

func (u *DesktopUI) newToolbarArea() *toolbarArea {
	i := kwidget.NewTappableImageWithMenu(icon.Characterplaceholder64Jpeg, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(ui.DefaultIconUnitSize))
	a := &toolbarArea{
		icon: i,
		name: widget.NewLabel(""),
		u:    u,
	}
	c := container.NewBorder(nil, nil, a.icon, nil, a.name)
	a.content = container.NewVBox(c, widget.NewSeparator())
	return a
}

func (a *toolbarArea) refresh() {
	c := a.u.CurrentCharacter()
	if c == nil {
		r, _ := fynetools.MakeAvatar(icon.Characterplaceholder64Jpeg)
		a.icon.SetResource(r)
		a.name.Text = "No character"
		a.name.TextStyle = fyne.TextStyle{Italic: true}
		a.name.Importance = widget.LowImportance
	} else {
		go a.u.UpdateAvatar(c.ID, func(r fyne.Resource) {
			a.icon.SetResource(r)
		})
		s := fmt.Sprintf("%s (%s)", c.EveCharacter.Name, c.EveCharacter.Corporation.Name)
		a.name.Text = s
		a.name.TextStyle = fyne.TextStyle{Bold: true}
		a.name.Importance = widget.MediumImportance
	}
	a.name.Refresh()
	a.icon.SetMenuItems(a.u.MakeCharacterSwitchMenu(a.icon.Refresh))
}
