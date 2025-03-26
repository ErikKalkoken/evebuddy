package desktopui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

type Toolbar struct {
	widget.BaseWidget

	icon *kwidget.TappableImage
	name *widget.Label
	u    app.UI
}

func NewToolbar(u app.UI) *Toolbar {
	i := kwidget.NewTappableImageWithMenu(icons.Characterplaceholder64Jpeg, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	a := &Toolbar{
		icon: i,
		name: widget.NewLabel(""),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *Toolbar) CreateRenderer() fyne.WidgetRenderer {
	x := container.NewBorder(nil, nil, a.icon, nil, a.name)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}

func (a *Toolbar) Update() {
	c := a.u.CurrentCharacter()
	if c == nil {
		r, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
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
