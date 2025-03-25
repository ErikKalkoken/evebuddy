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
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type solarSystemInfo struct {
	widget.BaseWidget

	iw *InfoWindow

	id            int32
	region        *kxwidget.TappableLabel
	constellation *kxwidget.TappableLabel
	logo          *canvas.Image
	name          *widget.Label
	security      *widget.Label
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw *InfoWindow, id int32) *solarSystemInfo {
	region := kxwidget.NewTappableLabel("", nil)
	region.Wrapping = fyne.TextWrapWord
	constellation := kxwidget.NewTappableLabel("", nil)
	constellation.Wrapping = fyne.TextWrapWord
	a := &solarSystemInfo{
		id:            id,
		region:        region,
		constellation: constellation,
		iw:            iw,
		logo:          makeInfoLogo(),
		name:          makeInfoName(),
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *solarSystemInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("solar system info update failed", "solarSystem", a.id, "error", err)
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
			widget.NewLabel("Solar System"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(colums, widget.NewLabel("Region"), a.region),
			container.New(colums, widget.NewLabel("Constellation"), a.constellation),
			container.New(colums, widget.NewLabel("Security"), a.security),
		))
	top := container.NewBorder(nil, nil, container.NewVBox(container.NewPadded(a.logo)), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *solarSystemInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	a.region.SetText(o.Constellation.Region.Name)
	a.region.OnTapped = func() {
		a.iw.ShowEveEntity(o.Constellation.Region.ToEveEntity())
	}
	a.constellation.SetText(o.Constellation.Name)
	a.constellation.OnTapped = func() {
		a.iw.ShowEveEntity(o.Constellation.ToEveEntity())
	}
	a.security.Text = o.SecurityStatusDisplay()
	a.security.Importance = o.SecurityType().ToImportance()
	a.security.Refresh()

	systemsLabel := widget.NewLabel("Loading...")
	systemsTab := container.NewTabItem("Stargates", systemsLabel)
	a.tabs.Append(systemsTab)

	planetsLabel := widget.NewLabel("Loading...")
	planetsTab := container.NewTabItem("Planets", planetsLabel)
	a.tabs.Append(planetsTab)

	stationsLabel := widget.NewLabel("Loading...")
	stationsTab := container.NewTabItem("Stations", stationsLabel)
	a.tabs.Append(stationsTab)

	structuresLabel := widget.NewLabel("Loading...")
	structuresTab := container.NewTabItem("Structures", structuresLabel)
	a.tabs.Append(structuresTab)

	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", fmt.Sprint(a.id))
		x.Action = func(v any) {
			a.iw.w.Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	a.tabs.Refresh()

	go func() {
		starID, planets, stargateIDs, stations, structures, err := a.iw.u.EveUniverseService().GetSolarSystemInfoESI(ctx, a.id)
		if err != nil {
			slog.Error("solar system info: Failed to load system info", "solarSystem", a.id, "error", err)
			stationsLabel.Text = ihumanize.Error(err)
			stationsLabel.Importance = widget.DangerImportance
			stationsLabel.Refresh()
			return

		}

		stationsTab.Content = NewEntityListFromEntities(a.iw.show, stations...)
		a.tabs.Refresh()

		oo := xslices.Map(structures, func(x *app.EveLocation) entityItem {
			return NewEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		})
		xx := NewEntityListFromItems(a.iw.show, oo...)
		note := widget.NewLabel("Only contains structures known through characters")
		note.Importance = widget.LowImportance
		structuresTab.Content = container.NewBorder(
			nil,
			note,
			nil,
			nil,
			xx,
		)
		a.tabs.Refresh()

		id, err := a.iw.u.EveUniverseService().GetStarTypeID(ctx, starID)
		if err != nil {
			return
		}
		r, err := a.iw.u.EveImageService().InventoryTypeIcon(id, app.IconPixelSize)
		if err != nil {
			slog.Error("solar system info: Failed to load logo", "solarSystem", a.id, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()

		go func() {
			ss, err := a.iw.u.EveUniverseService().GetSolarSystemsESI(ctx, stargateIDs)
			if err != nil {
				slog.Error("solar system info: Failed to load adjacent systems", "solarSystem", a.id, "error", err)
				systemsLabel.Text = ihumanize.Error(err)
				systemsLabel.Importance = widget.DangerImportance
				systemsLabel.Refresh()
				return
			}
			xx := xslices.Map(ss, NewEntityItemFromEveSolarSystem)
			systemsTab.Content = NewEntityListFromItems(a.iw.show, xx...)
			a.tabs.Refresh()
		}()

		go func() {
			pp, err := a.iw.u.EveUniverseService().GetSolarSystemPlanets(ctx, planets)
			if err != nil {
				slog.Error("solar system info: Failed to load planets", "solarSystem", a.id, "error", err)
				planetsLabel.Text = ihumanize.Error(err)
				planetsLabel.Importance = widget.DangerImportance
				planetsLabel.Refresh()
				return
			}
			xx := xslices.Map(pp, NewEntityItemFromEvePlanet)
			planetsTab.Content = NewEntityListFromItems(a.iw.show, xx...)
			a.tabs.Refresh()
		}()

	}()
	return nil
}
