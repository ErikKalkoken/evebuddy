package ui

import (
	"image/color"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/fynetools"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type pageBarCollection interface {
	NewPageBar(title string) *pageBar
}

type PageBarPage struct {
	widget.BaseWidget

	pb      *pageBar
	content fyne.CanvasObject
}

func NewPageBarPage(cpb pageBarCollection, title string, content fyne.CanvasObject) *PageBarPage {
	a := &PageBarPage{
		pb:      cpb.NewPageBar(title),
		content: content,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *PageBarPage) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		a.pb,
		nil,
		nil,
		nil,
		a.content,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *PageBarPage) SetTitle(text string) {
	a.pb.SetTitle(text)
}

type pageBar struct {
	widget.BaseWidget

	icon  *iwidget.TappableIcon
	title *widget.Label
}

func newPageBar(title string, icon fyne.Resource) *pageBar {
	i := iwidget.NewTappableIconWithMenu(icon, fyne.NewMenu(""))
	i.SetToolTip("Click to switch")
	l := widget.NewLabel(title)
	l.SizeName = theme.SizeNameSubHeadingText
	w := &pageBar{
		icon:  i,
		title: l,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *pageBar) SetTitle(text string) {
	w.title.SetText(text)
}

func (w *pageBar) SetIcon(r fyne.Resource) {
	w.icon.SetResource(r)
}

func (w *pageBar) SetMenu(items []*fyne.MenuItem) {
	w.icon.SetMenuItems(items)
}

func (w *pageBar) CreateRenderer() fyne.WidgetRenderer {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSquareSize(w.title.MinSize().Height - theme.Padding()))
	c := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewPadded(container.NewCenter(container.NewStack(spacer, w.icon))),
		w.title,
	)
	return widget.NewSimpleRenderer(c)
}

type pageBarCollectionForCharacter struct {
	bars         []*pageBar
	fallbackIcon fyne.Resource
	u            *DesktopUI
}

func NewPageBarCollectionForCharacters(u *DesktopUI) *pageBarCollectionForCharacter {
	fallback := icons.Characterplaceholder64Jpeg
	icon, err := fynetools.MakeAvatar(fallback)
	if err != nil {
		slog.Error("failed to make avatar", "error", err)
		icon = fallback
	}
	c := &pageBarCollectionForCharacter{
		bars:         make([]*pageBar, 0),
		fallbackIcon: icon,
		u:            u,
	}
	return c
}

func (pbc *pageBarCollectionForCharacter) NewPageBar(title string) *pageBar {
	pb := newPageBar(title, pbc.fallbackIcon)
	pbc.bars = append(pbc.bars, pb)
	return pb
}

func (pbc *pageBarCollectionForCharacter) update() {
	if !pbc.u.hasCharacter() {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.SetIcon(pbc.fallbackIcon)
			})
		}
		return
	}
	go pbc.u.updateCharacterAvatar(pbc.u.currentCharacterID(), func(r fyne.Resource) {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.SetIcon(r)
			})
		}
	})
	items := pbc.u.makeCharacterSwitchMenu(func() {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.Refresh()
			})
		}
	})
	for _, pb := range pbc.bars {
		fyne.Do(func() {
			pb.SetMenu(items)
		})
	}
}

type pageBarCollectionForCorporation struct {
	bars         []*pageBar
	fallbackIcon fyne.Resource
	u            *DesktopUI
}

func NewPageBarCollectionForCorporations(u *DesktopUI) *pageBarCollectionForCorporation {
	fallback := icons.Corporationplaceholder64Png
	icon, err := fynetools.MakeAvatar(fallback)
	if err != nil {
		slog.Error("failed to make avatar", "error", err)
		icon = fallback
	}
	c := &pageBarCollectionForCorporation{
		bars:         make([]*pageBar, 0),
		fallbackIcon: icon,
		u:            u,
	}
	return c
}

func (pbc *pageBarCollectionForCorporation) NewPageBar(title string) *pageBar {
	pb := newPageBar(title, pbc.fallbackIcon)
	pbc.bars = append(pbc.bars, pb)
	return pb
}

func (pbc *pageBarCollectionForCorporation) update() {
	if !pbc.u.hasCorporation() {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.SetIcon(pbc.fallbackIcon)
			})
		}
		return
	}
	go pbc.u.updateCorporationAvatar(pbc.u.currentCorporationID(), func(r fyne.Resource) {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.SetIcon(r)
			})
		}
	})
	items := pbc.u.makeCorporationSwitchMenu(func() {
		for _, pb := range pbc.bars {
			fyne.Do(func() {
				pb.Refresh()
			})
		}
	})
	for _, pb := range pbc.bars {
		fyne.Do(func() {
			pb.SetMenu(items)
		})
	}
}
