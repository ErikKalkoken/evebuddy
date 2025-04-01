package desktopui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type desktopUI interface {
	app.UI
	ShowSettingsWindow()
	ShowAboutDialog()
}

type Toolbar struct {
	widget.BaseWidget

	search   *widget.Entry
	icon     *kwidget.TappableImage
	about    *iwidget.IconButton
	settings *iwidget.IconButton
	u        desktopUI
}

func NewToolbar(u desktopUI) *Toolbar {
	i := kwidget.NewTappableImageWithMenu(icons.Characterplaceholder64Jpeg, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	search := widget.NewEntry()
	search.PlaceHolder = "Search New Eden"
	search.Scroll = container.ScrollNone
	search.Wrapping = fyne.TextWrapOff
	a := &Toolbar{
		icon:     i,
		u:        u,
		search:   search,
		about:    iwidget.NewIconButton(theme.InfoIcon(), u.ShowAboutDialog),
		settings: iwidget.NewIconButton(theme.NewThemedResource(icons.CogSvg), u.ShowSettingsWindow),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *Toolbar) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	x := container.NewGridWithColumns(
		3,
		container.NewHBox(),
		a.search,
		container.New(layout.NewCustomPaddedHBoxLayout(2*p), layout.NewSpacer(), a.about, a.settings, a.icon),
	)
	c := container.NewVBox(x, widget.NewSeparator())
	return widget.NewSimpleRenderer(c)
}

func (a *Toolbar) Update() {
	if c := a.u.CurrentCharacter(); c == nil {
		r, _ := fynetools.MakeAvatar(icons.Characterplaceholder64Jpeg)
		a.icon.SetResource(r)
	} else {
		go a.u.UpdateAvatar(c.ID, func(r fyne.Resource) {
			a.icon.SetResource(r)
		})
	}
	a.icon.SetMenuItems(a.u.MakeCharacterSwitchMenu(a.icon.Refresh))
}
