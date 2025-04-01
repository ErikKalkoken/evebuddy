package infowindow

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

	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type regionInfo struct {
	widget.BaseWidget

	id int32
	iw *InfoWindow

	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
}

func newRegionInfo(iw *InfoWindow, id int32) *regionInfo {
	a := &regionInfo{
		iw:   iw,
		id:   id,
		logo: makeInfoLogo(),
		name: makeInfoName(),
		tabs: container.NewAppTabs(),
	}
	a.logo.Resource = icons.Region64Png
	a.ExtendBaseWidget(a)
	return a
}

func (a *regionInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("region info update failed", "solarSystem", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.ErrorDisplay(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *regionInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)

	desc := widget.NewLabel(o.DescriptionPlain())
	desc.Wrapping = fyne.TextWrapWord
	a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(desc)))
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.iw.w.Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	cLabel := widget.NewLabel("Loading...")
	constellations := container.NewTabItem("Constellations", cLabel)
	a.tabs.Append(constellations)
	a.tabs.Select(constellations)
	a.tabs.Refresh()
	go func() {
		oo, err := a.iw.u.EveUniverseService().GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			slog.Error("region info: Failed to load constellations", "region", o.ID, "error", err)
			cLabel.Text = a.iw.u.ErrorDisplay(err)
			cLabel.Importance = widget.DangerImportance
			cLabel.Refresh()
			return
		}
		xx := xslices.Map(oo, NewEntityItemFromEveEntity)
		constellations.Content = NewEntityListFromItems(a.iw.show, xx...)
		a.tabs.Refresh()
		a.tabs.Refresh()
	}()
	return nil
}
