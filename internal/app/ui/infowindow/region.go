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

	description    *widget.Label
	constellations *entityList
	id             int32
	iw             *InfoWindow
	logo           *canvas.Image
	name           *widget.Label
	tabs           *container.AppTabs
}

func newRegionInfo(iw *InfoWindow, id int32) *regionInfo {
	description := widget.NewLabel("")
	description.Wrapping = fyne.TextWrapWord
	a := &regionInfo{
		iw:          iw,
		id:          id,
		description: description,
		logo:        makeInfoLogo(),
		name:        makeInfoName(),
		tabs:        container.NewAppTabs(),
	}
	a.logo.Resource = icons.Region64Png
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
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("region info update failed", "solarSystem", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
		}
	}()
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
				a.iw.makeZkillboardIcon(a.id, infoRegion),
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

func (a *regionInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateRegionESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
		a.name.SetText(o.Name)
		a.description.SetText(o.DescriptionPlain())
	})
	fyne.Do(func() {
		if !a.iw.u.IsDeveloperMode() {
			return
		}
		x := newAttributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	})
	go func() {
		oo, err := a.iw.u.EveUniverseService().GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			slog.Error("region info: Failed to load constellations", "region", o.ID, "error", err)
			return
		}
		items := xslices.Map(oo, NewEntityItemFromEveEntity)
		fyne.Do(func() {
			a.constellations.set(items...)
		})
	}()
	return nil
}
