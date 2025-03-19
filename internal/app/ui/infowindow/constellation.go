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
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type constellationArea struct {
	Content fyne.CanvasObject

	iw InfoWindow
	w  fyne.Window

	region *kxwidget.TappableLabel
	logo   *canvas.Image
	name   *widget.Label
	tabs   *container.AppTabs
}

func newConstellationArea(iw InfoWindow, constellationID int32, w fyne.Window) *constellationArea {
	region := kxwidget.NewTappableLabel("", nil)
	region.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	s := float32(app.IconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icons.Region64Png, fyne.NewSquareSize(s))
	a := &constellationArea{
		iw:     iw,
		logo:   logo,
		name:   name,
		region: region,
		tabs:   container.NewAppTabs(),
		w:      w,
	}
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
	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(constellationID)
		if err != nil {
			slog.Error("constellation info update failed", "solarSystem", constellationID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *constellationArea) load(constellationID int32) error {
	ctx := context.Background()
	o, err := a.iw.eus.GetOrCreateEveConstellationESI(ctx, constellationID)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	a.region.SetText(o.Region.Name)
	a.region.OnTapped = func() {
		a.iw.ShowEveEntity(o.Region.ToEveEntity())
	}

	if a.iw.isDeveloperMode {
		x := NewAtributeItem("EVE ID", fmt.Sprint(o.ID))
		x.Action = func(v any) {
			a.w.Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	sLabel := widget.NewLabel("Loading...")
	solarSystems := container.NewTabItem("Solar Systems", sLabel)
	a.tabs.Append(solarSystems)
	a.tabs.Refresh()
	go func() {
		oo, err := a.iw.eus.GetConstellationSolarSytemsESI(ctx, o.ID)
		if err != nil {
			slog.Error("constellation info: Failed to load constellations", "region", o.ID, "error", err)
			sLabel.Text = ihumanize.Error(err)
			sLabel.Importance = widget.DangerImportance
			sLabel.Refresh()
			return
		}
		xx := slices.Collect(xiter.MapSlice(oo, NewEntityItemFromEveSolarSystem))
		solarSystems.Content = NewEntityListFromItems(a.iw.Show, xx...)
		a.tabs.Refresh()
	}()
	return nil
}
