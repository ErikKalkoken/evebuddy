package infoviewer

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
)

type factionInfo struct {
	widget.BaseWidget
	baseInfo

	id          int64
	logo        *canvas.Image
	tabs        *container.AppTabs
	description *widget.Label
}

func newFactionInfo(iw *InfoViewer, id int64) *factionInfo {
	a := &factionInfo{
		description: newLabelWithWrapAndSelectable(""),
		id:          id,
		logo:        makeInfoLogo(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.BlankSvg
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Description", container.NewVScroll(a.description)),
	)
	return a
}

func (a *factionInfo) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *factionInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateFactionESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.Description)
		a.tabs.Refresh()
	})
	fyne.Do(func() {
		a.iw.u.EVEImage().FactionLogoAsync(a.id, ui.IconPixelSize, func(r fyne.Resource) {
			a.logo.Resource = r
			a.logo.Refresh()
		})
	})
	items := []attributeItem{
		newAttributeItem("Unique", o.IsUnique),
		newAttributeItem("Stations", o.StationCount),
		newAttributeItem("Systems with station", o.StationSystemCount),
	}
	if v, ok := o.Corporation.Value(); ok {
		items = append(items, newAttributeItem("Corporation", v))
	}
	if v, ok := o.MilitiaCorporation.Value(); ok {
		items = append(items, newAttributeItem("Military Corporation", v))
	}
	if v, ok := o.SolarSystem.Value(); ok {
		items = append(items, newAttributeItem("Solar System", &app.EveEntity{
			Category: app.EveEntitySolarSystem,
			ID:       v.ID,
			Name:     v.Name,
		}))
	}
	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			fyne.CurrentApp().Clipboard().SetContent(v.(string))
		}
		items = append(items, x)
	}
	attributeList := newAttributeList(a.iw, items...)
	attributesTab := container.NewTabItem("Attributes", attributeList)
	fyne.Do(func() {
		a.tabs.Append(attributesTab)
	})
	return nil
}
