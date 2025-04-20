package ui

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

type PageBar struct {
	widget.BaseWidget

	buttons []*widget.Button
	icon    *kwidget.TappableImage
	title   *iwidget.Label
	u       *BaseUI
}

func newPageBar(title string, icon fyne.Resource, u *BaseUI, buttons ...*widget.Button) *PageBar {
	i := kwidget.NewTappableImageWithMenu(icon, fyne.NewMenu(""))
	i.SetFillMode(canvas.ImageFillContain)
	i.SetMinSize(fyne.NewSquareSize(app.IconUnitSize))
	w := &PageBar{
		buttons: buttons,
		icon:    i,
		title:   iwidget.NewLabelWithSize(title, theme.SizeNameSubHeadingText),
		u:       u,
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
	box := container.NewHBox(w.title, layout.NewSpacer())
	if len(w.buttons) > 0 {
		for _, b := range w.buttons {
			box.Add(container.NewCenter(b))
		}
	}
	box.Add(container.NewCenter(w.icon))
	return widget.NewSimpleRenderer(box)
}

type PageBarCollection struct {
	bars         []*PageBar
	fallbackIcon fyne.Resource
	u            *DesktopUI
}

func NewPageBarCollection(u *DesktopUI) *PageBarCollection {
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

func (c *PageBarCollection) NewPageBar(title string, buttons ...*widget.Button) *PageBar {
	pb := newPageBar(title, c.fallbackIcon, c.u.BaseUI, buttons...)
	c.bars = append(c.bars, pb)
	return pb
}

func (c *PageBarCollection) update() {
	if !c.u.HasCharacter() {
		for _, pb := range c.bars {
			fyne.Do(func() {
				pb.SetIcon(c.fallbackIcon)
			})
		}
		return
	}
	go c.u.updateAvatar(c.u.CurrentCharacterID(), func(r fyne.Resource) {
		for _, pb := range c.bars {
			fyne.Do(func() {
				pb.SetIcon(r)
			})
		}
	})
	items := c.u.makeCharacterSwitchMenu(func() {
		fyne.Do(func() {
			for _, pb := range c.bars {
				pb.Refresh()
			}
		})
	})
	for _, pb := range c.bars {
		fyne.Do(func() {
			pb.SetMenu(items)
		})
	}
}
