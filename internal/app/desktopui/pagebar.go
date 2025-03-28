package desktopui

import (
	"log/slog"

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

type PageBarCollection struct {
	bars         []*PageBar
	fallbackIcon fyne.Resource
	u            app.UI
}

func NewPageBarCollection(u app.UI) *PageBarCollection {
	fallback := icons.Characterplaceholder64Jpeg
	icon, err := fynetools.MakeAvatar(fallback)
	if err != nil {
		slog.Error("failed to make avatar", "error", err)
		icon = fallback
	}
	c := &PageBarCollection{
		bars:         make([]*PageBar, 0),
		fallbackIcon: icon,
		u:            u,
	}
	return c
}

func (c *PageBarCollection) NewPageBar(title string) *PageBar {
	pb := newPageBar(title, c.fallbackIcon, c.u)
	c.bars = append(c.bars, pb)
	return pb
}

func (c *PageBarCollection) Update() {
	if !c.u.HasCharacter() {
		for _, pb := range c.bars {
			pb.SetIcon(c.fallbackIcon)
		}
		return
	}
	go c.u.UpdateAvatar(c.u.CurrentCharacterID(), func(r fyne.Resource) {
		for _, pb := range c.bars {
			pb.SetIcon(r)
		}
	})
	items := c.u.MakeCharacterSwitchMenu(func() {
		for _, pb := range c.bars {
			pb.Refresh()
		}
	})
	for _, pb := range c.bars {
		pb.SetMenu(items)
	}
}

type PageBar struct {
	widget.BaseWidget

	icon  *kwidget.TappableImage
	title *iwidget.Label
	u     app.UI
}

func newPageBar(title string, icon fyne.Resource, u app.UI) *PageBar {
	i := kwidget.NewTappableImageWithMenu(icon, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w := &PageBar{
		icon:  i,
		title: iwidget.NewLabelWithSize(title, theme.SizeNameSubHeadingText),
		u:     u,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PageBar) SetIcon(r fyne.Resource) {
	w.icon.SetResource(r)
}

func (w *PageBar) SetMenu(items []*fyne.MenuItem) {
	w.icon.SetMenuItems(items)
}

func (w *PageBar) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox(
		container.NewVBox(layout.NewSpacer(), w.title, layout.NewSpacer()),
		layout.NewSpacer(),
		container.NewPadded(w.icon),
	)
	return widget.NewSimpleRenderer(c)
}
