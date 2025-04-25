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
)

type solarSystemInfo struct {
	widget.BaseWidget

	constellation *kxwidget.TappableLabel
	id            int32
	iw            *InfoWindow
	logo          *canvas.Image
	name          *widget.Label
	planets       *entityList
	region        *kxwidget.TappableLabel
	security      *widget.Label
	stargates     *entityList
	stations      *entityList
	structures    *entityList
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
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("solar system info update failed", "solarSystem", a.id, "error", err)
			fyne.Do(func() {
				a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", a.iw.u.ErrorDisplay(err))
				a.name.Importance = widget.DangerImportance
				a.name.Refresh()
			})
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
				a.iw.makeZkillboardIcon(a.id, infoSolarSystem),
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

func (a *solarSystemInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateSolarSystemESI(ctx, a.id)
	if err != nil {
		return err
	}
	fyne.Do(func() {
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
	})

	if a.iw.u.IsDeveloperMode() {
		x := newAttributeItem("EVE ID", fmt.Sprint(a.id))
		x.Action = func(v any) {
			a.iw.u.App().Clipboard().SetContent(v.(string))
		}
		attributeList := newAttributeList(a.iw, []attributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		fyne.Do(func() {
			a.tabs.Append(attributesTab)
		})
	}

	a.tabs.Refresh()

	go func() {
		starID, planets, stargateIDs, stations, structures, err := a.iw.u.EveUniverseService().GetSolarSystemInfoESI(ctx, a.id)
		if err != nil {
			slog.Error("solar system info: Failed to load system info", "solarSystem", a.id, "error", err)
			return
		}
		items := entityItemsFromEveEntities(stations)
		fyne.Do(func() {
			a.stations.set(items...)
		})
		oo := xslices.Map(structures, func(x *app.EveLocation) entityItem {
			return newEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		})
		fyne.Do(func() {
			a.structures.set(oo...)
		})

		id, err := a.iw.u.EveUniverseService().GetStarTypeID(ctx, starID)
		if err != nil {
			return
		}
		r, err := a.iw.u.EveImageService().InventoryTypeIcon(id, app.IconPixelSize)
		if err != nil {
			slog.Error("solar system info: Failed to load logo", "solarSystem", a.id, "error", err)
			return
		}
		fyne.Do(func() {
			a.logo.Resource = r
			a.logo.Refresh()
		})

		go func() {
			ss, err := a.iw.u.EveUniverseService().GetStargateSolarSystemsESI(ctx, stargateIDs)
			if err != nil {
				slog.Error("solar system info: Failed to load adjacent systems", "solarSystem", a.id, "error", err)
				return
			}
			items := xslices.Map(ss, newEntityItemFromEveSolarSystem)
			fyne.Do(func() {
				a.stargates.set(items...)
			})
		}()

		go func() {
			pp, err := a.iw.u.EveUniverseService().GetSolarSystemPlanets(ctx, planets)
			if err != nil {
				slog.Error("solar system info: Failed to load planets", "solarSystem", a.id, "error", err)
				return
			}
			items := xslices.Map(pp, newEntityItemFromEvePlanet)
			fyne.Do(func() {
				a.planets.set(items...)
			})
		}()

	}()
	return nil
}
