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
	"golang.org/x/sync/errgroup"

	"github.com/ErikKalkoken/evebuddy/internal/app"
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

func newSolarSystemInfo(iw *InfoWindow, id int64) *solarSystemInfo {
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
		container.NewTabItem("Stargates", a.stargates),
		container.NewTabItem("Planets", a.planets),
		container.NewTabItem("Stations", a.stations),
		container.NewTabItem("Structures", container.NewBorder(
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
				layout.NewCustomPaddedHBoxLayout(3*p),
				layout.NewSpacer(),
				a.iw.makeZKillboardIcon(a.id, infoSolarSystem),
				a.iw.makeDotlanIcon(a.id, infoSolarSystem),
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
	o, err := a.iw.eus.GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	starID, planets, stargateIDs, stations, structures, err := a.iw.eus.GetSolarSystemInfoESI(ctx, a.id)
	if err != nil {
		return err
	}
	g := new(errgroup.Group)
	g.Go(func() error {
		if a.iw.u.IsDeveloperMode() {
			x := newAttributeItem("EVE ID", fmt.Sprint(a.id))
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
			a.region.SetText(o.Constellation.Region.Name)
			a.region.OnTapped = func() {
				a.iw.showEveEntity(o.Constellation.Region.EveEntity())
			}
			a.constellation.SetText(o.Constellation.Name)
			a.constellation.OnTapped = func() {
				a.iw.showEveEntity(o.Constellation.EveEntity())
			}
			a.security.Text = o.SecurityStatusDisplay()
			a.security.Importance = o.SecurityType().ToImportance()
			a.security.Refresh()
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		stationItems := entityItemsFromEveEntities(stations)
		structureItems := xslices.Map(structures, func(x *app.EveLocation) entityItem {
			return newEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		})
		fyne.Do(func() {
			a.stations.set(stationItems...)
			a.structures.set(structureItems...)
			a.tabs.Refresh()
		})
		return nil
	})
	if v, ok := starID.Value(); ok {
		g.Go(func() error {
			id, err := a.iw.eus.GetStarTypeID(ctx, v)
			if err != nil {
				return err
			}
			fyne.Do(func() {
				a.iw.eis.InventoryTypeIconAsync(id, app.IconPixelSize, func(r fyne.Resource) {
					a.logo.Resource = r
					a.logo.Refresh()
				})
			})
			return nil
		})
	}
	g.Go(func() error {
		ss, err := a.iw.eus.GetStargatesSolarSystemsESI(ctx, stargateIDs)
		if err != nil {
			return err
		}
		items := xslices.Map(ss, newEntityItemFromEveSolarSystem)
		fyne.Do(func() {
			a.stargates.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	g.Go(func() error {
		pp, err := a.iw.eus.GetSolarSystemPlanets(ctx, planets)
		if err != nil {
			return err
		}
		items := xslices.Map(pp, newEntityItemFromEvePlanet)
		fyne.Do(func() {
			a.planets.set(items...)
			a.tabs.Refresh()
		})
		return nil
	})
	return g.Wait()
}
