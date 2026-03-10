package infowindow

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type regionInfo struct {
	widget.BaseWidget
	baseInfo

	description    *widget.Label
	constellations *entityList
	id             int64
	logo           *canvas.Image
	tabs           *container.AppTabs
}

func newRegionInfo(iw *InfoWindow, id int64) *regionInfo {
	a := &regionInfo{
		id:          id,
		description: newLabelWithWrapAndSelectable(""),
		logo:        makeInfoLogo(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.Region64Png
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.constellations = newEntityList(a.iw.show)
	constellations := container.NewTabItem("Constellations", a.constellations)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
		constellations,
	)
	a.tabs.Select(constellations)
	return a
}

func (a *regionInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
	)
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoRegion),
				a.iw.makeDotlanIcon(a.id, infoRegion),
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *regionInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if !a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			fyne.Do(func() {
				a.tabs.Append(attributesTab)
			})
		}
		fyne.Do(func() {
			a.name.SetText(o.Name)
			a.description.SetText(o.DescriptionPlain())
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		oo, err := a.iw.u.EVEUniverse().GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			return err
		}
		items := xslices.Map(oo, newEntityItemFromEveEntity)
		fyne.Do(func() {
			a.constellations.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	return nil
}
