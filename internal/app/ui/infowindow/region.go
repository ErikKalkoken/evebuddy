package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type regionInfo struct {
	widget.BaseWidget

	id int32
	iw InfoWindow
	w  fyne.Window

	logo *canvas.Image
	name *widget.Label
	tabs *container.AppTabs
}

func newRegionInfo(iw InfoWindow, regionID int32, w fyne.Window) *regionInfo {
	name := widget.NewLabel("")
	name.Wrapping = fyne.TextWrapWord
	s := float32(app.IconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icons.Region64Png, fyne.NewSquareSize(s))
	a := &regionInfo{
		iw:   iw,
		id:   regionID,
		logo: logo,
		name: name,
		tabs: container.NewAppTabs(),
		w:    w,
	}
	a.ExtendBaseWidget(a)
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
	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(a.id)
		if err != nil {
			slog.Error("region info update failed", "solarSystem", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return widget.NewSimpleRenderer(c)
}

func (a *regionInfo) load(regionID int32) error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateRegionESI(ctx, regionID)
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
			a.w.Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	cLabel := widget.NewLabel("Loading...")
	constellations := container.NewTabItem("Constellations", cLabel)
	a.tabs.Append(constellations)
	a.tabs.Refresh()
	go func() {
		oo, err := a.iw.u.EveUniverseService().GetRegionConstellationsESI(ctx, o.ID)
		if err != nil {
			slog.Error("region info: Failed to load constellations", "region", o.ID, "error", err)
			cLabel.Text = ihumanize.Error(err)
			cLabel.Importance = widget.DangerImportance
			cLabel.Refresh()
			return
		}
		xx := slices.Collect(xiter.MapSlice(oo, NewEntityItemFromEveEntity))
		constellations.Content = NewEntityListFromItems(a.iw.show, xx...)
		a.tabs.Refresh()
		a.tabs.Refresh()
	}()
	return nil
}
