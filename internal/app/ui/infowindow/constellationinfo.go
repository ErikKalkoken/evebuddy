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
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"

	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type constellationInfo struct {
	widget.BaseWidget
	baseInfo

	id      int64
	logo    *canvas.Image
	region  *widget.Hyperlink
	systems *entityList
	tabs    *container.AppTabs
}

func newConstellationInfo(iw *InfoWindow, id int64) *constellationInfo {
	region := widget.NewHyperlink("", nil)
	region.Wrapping = fyne.TextWrapWord
	a := &constellationInfo{

		id:     id,
		logo:   makeInfoLogo(),
		region: region,
		tabs:   container.NewAppTabs(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.logo.Resource = icons.Constellation64Png
	a.systems = newEntityList(a.iw.show)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Solar Systems", a.systems),
	)
	return a
}

func (a *constellationInfo) CreateRenderer() fyne.WidgetRenderer {
	columns := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(columns, widget.NewLabel("Region"), a.region),
		))
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *constellationInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateConstellationESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.region.SetText(o.Region.Name)
		a.region.OnTapped = func() {
			a.iw.Show(o.Region.EveEntity())
		}

		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
			x.Action = func(v any) {
				fyne.CurrentApp().Clipboard().SetContent(v.(string))
			}
			attributeList := newAttributeList(a.iw, []attributeItem{x}...)
			attributesTab := container.NewTabItem("Attributes", attributeList)
			a.tabs.Append(attributesTab)
			a.tabs.Refresh()
		}
	})
	oo, err := a.iw.u.EVEUniverse().GetConstellationSolarSystemsESI(ctx, o.ID)
	if err != nil {
		return err
	}
	xx := xslices.Map(oo, newEntityItemFromEveSolarSystem)
	fyne.Do(func() {
		a.systems.set(xx...)
		a.tabs.Refresh()
	})
	return nil
}
