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
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type constellationInfo struct {
	widget.BaseWidget

	iw *InfoWindow

	id     int32
	region *kxwidget.TappableLabel
	logo   *canvas.Image
	name   *widget.Label
	tabs   *container.AppTabs
}

func newConstellationInfo(iw *InfoWindow, id int32) *constellationInfo {
	region := kxwidget.NewTappableLabel("", nil)
	region.Wrapping = fyne.TextWrapWord
	a := &constellationInfo{
		iw:     iw,
		id:     id,
		logo:   makeInfoLogo(),
		name:   makeInfoName(),
		region: region,
		tabs:   container.NewAppTabs(),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *constellationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("constellation info update failed", "solarSystem", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	colums := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Region"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(colums, widget.NewLabel("Region"), a.region),
		))
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *constellationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateConstellationESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	a.region.SetText(o.Region.Name)
	a.region.OnTapped = func() {
		a.iw.ShowEveEntity(o.Region.ToEveEntity())
	}

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

	sLabel := widget.NewLabel("Loading...")
	solarSystems := container.NewTabItem("Solar Systems", sLabel)
	a.tabs.Append(solarSystems)
	a.tabs.Select(solarSystems)
	a.tabs.Refresh()
	go func() {
		oo, err := a.iw.u.EveUniverseService().GetConstellationSolarSytemsESI(ctx, o.ID)
		if err != nil {
			slog.Error("constellation info: Failed to load constellations", "region", o.ID, "error", err)
			sLabel.Text = ihumanize.Error(err)
			sLabel.Importance = widget.DangerImportance
			sLabel.Refresh()
			return
		}
		xx := xslices.Map(oo, NewEntityItemFromEveSolarSystem)
		solarSystems.Content = NewEntityListFromItems(a.iw.show, xx...)
		a.tabs.Refresh()
	}()
	return nil
}
