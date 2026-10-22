package infoviewer

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type solarSystemInfo struct {
	widget.BaseWidget
	baseInfo

	constellation *widget.Hyperlink
	id            int64
	logo          *canvas.Image
	planets       *entityList
	region        *widget.Hyperlink
	security      *widget.Label
	stargates     *entityList
	stations      *entityList
	structures    *entityList
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw *InfoViewer, id int64) *solarSystemInfo {
	region := widget.NewHyperlink("", nil)
	region.Wrapping = fyne.TextWrapWord
	constellation := widget.NewHyperlink("", nil)
	constellation.Wrapping = fyne.TextWrapWord
	a := &solarSystemInfo{
		id:            id,
		region:        region,
		constellation: constellation,
		logo:          makeInfoLogo(),
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
	}
	a.initBase(iw)
	a.ExtendBaseWidget(a)
	a.planets = newEntityList(a.iw.show)
	a.stargates = newEntityList(a.iw.show)
	a.stations = newEntityList(a.iw.show)
	a.structures = newEntityList(a.iw.show)
	note := widget.NewLabel("Only contains structures known through characters")
	note.Importance = widget.LowImportance
	a.tabs = container.NewAppTabs(
		ui.NewTabItem("Stargates", a.stargates),
		ui.NewTabItem("Planets", a.planets),
		ui.NewTabItem("Stations", a.stations),
		ui.NewTabItem("Structures", container.NewBorder(
			nil,
			note,
			nil,
			nil,
			a.structures,
		)),
	)
	return a
}

func (a *solarSystemInfo) CreateRenderer() fyne.WidgetRenderer {
	columns := kxlayout.NewColumns(120)
	p := theme.Padding()
	main := container.NewVBox(
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			widget.NewLabel("Solar System"),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			container.New(columns, widget.NewLabel("Region"), a.region),
			container.New(columns, widget.NewLabel("Constellation"), a.constellation),
			container.New(columns, widget.NewLabel("Security"), a.security),
		),
	)
	top := container.NewBorder(
		nil,
		nil,
		container.NewVBox(
			container.NewPadded(a.logo),
			container.New(
				layout.NewCustomPaddedHBoxLayout(linkIconPaddingFactor*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, SolarSystem),
				a.iw.makeDotlanIcon(a.id, SolarSystem),
				layout.NewSpacer(),
			),
		),
		nil,
		main,
	)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *solarSystemInfo) update(ctx context.Context) error {
	o, err := a.iw.u.EVEUniverse().GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	starID, planets, stargateIDs, stations, structures, err := a.iw.u.EVEUniverse().GetSolarSystemInfoESI(ctx, a.id)
	if err != nil {
		return err
	}
	stationItems := entityItemsFromEveEntities(stations)
	structureItems := xslices.Map(structures, func(x *app.EveLocation) entityItem {
		return newEntityItem(
			x.ID,
			x.Name,
			"Structure",
			Location,
		)
	})
	var starTypeID int64
	var stargateItems, planetItems []entityItem
	g := new(errgroup.Group)
	if v, ok := starID.Value(); ok {
		g.Go(func() error {
			id, err := a.iw.u.EVEUniverse().GetStarTypeID(ctx, v)
			if ctx.Err() != nil {
				return nil // page closed or shutting down
			}
			if err != nil {
				// only needed for the logo
				slog.Error("Failed to fetch star type for solar system", "solarSystemID", a.id, "error", err)
				return nil
			}
			starTypeID = id
			return nil
		})
	}
	g.Go(func() error {
		ss, err := a.iw.u.EVEUniverse().GetStargatesSolarSystemsESI(ctx, stargateIDs)
		if err != nil {
			return err
		}
		stargateItems = xslices.Map(ss, newEntityItemFromEveSolarSystem)
		return nil
	})
	g.Go(func() error {
		pp, err := a.iw.u.EVEUniverse().GetSolarSystemPlanets(ctx, planets)
		if err != nil {
			return err
		}
		planetItems = xslices.Map(pp, newEntityItemFromEvePlanet)
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	fyne.Do(func() {
		if a.iw.u.IsDeveloperMode() {
			a.tabs.Append(ui.NewTabItem("Attributes", newAttributeList(a.iw, newEVEIDAttributeItem(a.id))))
		}
		a.name.SetText(o.Name)
		a.region.SetText(o.Constellation.Region.Name)
		a.region.OnTapped = func() {
			a.iw.Show(o.Constellation.Region.ToEveEntity())
		}
		a.constellation.SetText(o.Constellation.Name)
		a.constellation.OnTapped = func() {
			a.iw.Show(o.Constellation.ToEveEntity())
		}
		a.security.Text = o.SecurityStatusDisplay()
		a.security.Importance = o.SecurityType().ToImportance()
		a.security.Refresh()
		if starTypeID != 0 {
			a.iw.u.EVEImage().InventoryTypeIconAsync(starTypeID, ui.IconPixelSize, func(r fyne.Resource) {
				a.logo.Resource = r
				a.logo.Refresh()
			})
		}
		a.stations.set(stationItems...)
		a.structures.set(structureItems...)
		a.stargates.set(stargateItems...)
		a.planets.set(planetItems...)
		a.tabs.Refresh()
	})
	return nil
}
