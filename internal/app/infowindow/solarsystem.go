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

type solarSystemInfo struct {
	widget.BaseWidget

	iw InfoWindow
	w  fyne.Window

	id            int32
	region        *kxwidget.TappableLabel
	constellation *kxwidget.TappableLabel
	logo          *canvas.Image
	name          *widget.Label
	security      *widget.Label
	tabs          *container.AppTabs
}

func newSolarSystemInfo(iw InfoWindow, id int32, w fyne.Window) *solarSystemInfo {
	region := kxwidget.NewTappableLabel("", nil)
	region.Wrapping = fyne.TextWrapWord
	constellation := kxwidget.NewTappableLabel("", nil)
	constellation.Wrapping = fyne.TextWrapWord
	name := widget.NewLabel("")
	name.Wrapping = fyne.TextWrapWord
	name.TextStyle.Bold = true
	s := float32(app.IconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(s))
	a := &solarSystemInfo{
		id:            id,
		region:        region,
		constellation: constellation,
		iw:            iw,
		logo:          logo,
		name:          name,
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
		w:             w,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *solarSystemInfo) CreateRenderer() fyne.WidgetRenderer {
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
	top := container.NewBorder(nil, nil, container.NewVBox(a.logo), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(a.id)
		if err != nil {
			slog.Error("solar system info update failed", "solarSystem", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return widget.NewSimpleRenderer(c)
}

func (a *solarSystemInfo) load(solarSystemID int32) error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateSolarSystemESI(ctx, solarSystemID)
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
		x := NewAtributeItem("EVE ID", fmt.Sprint(solarSystemID))
		x.Action = func(v any) {
			a.w.Clipboard().SetContent(v.(string))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	a.tabs.Refresh()

	go func() {
		starID, planets, stargateIDs, stations, structures, err := a.iw.u.EveUniverseService().GetSolarSystemInfoESI(ctx, solarSystemID)
		if err != nil {
			slog.Error("solar system info: Failed to load system info", "solarSystem", solarSystemID, "error", err)
			stationsLabel.Text = ihumanize.Error(err)
			stationsLabel.Importance = widget.DangerImportance
			stationsLabel.Refresh()
			return

		}

		stationsTab.Content = NewEntityListFromEntities(a.iw.show, stations...)
		a.tabs.Refresh()

		oo := slices.Collect(xiter.MapSlice(structures, func(x *app.EveLocation) entityItem {
			return NewEntityItem(
				x.ID,
				x.Name,
				"Structure",
				infoLocation,
			)
		}))
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
			slog.Error("solar system info: Failed to load logo", "solarSystem", solarSystemID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()

		go func() {
			ss, err := a.iw.u.EveUniverseService().GetSolarSystemsESI(ctx, stargateIDs)
			if err != nil {
				slog.Error("solar system info: Failed to load adjacent systems", "solarSystem", solarSystemID, "error", err)
				systemsLabel.Text = ihumanize.Error(err)
				systemsLabel.Importance = widget.DangerImportance
				systemsLabel.Refresh()
				return
			}
			xx := slices.Collect(xiter.MapSlice(ss, NewEntityItemFromEveSolarSystem))
			systemsTab.Content = NewEntityListFromItems(a.iw.show, xx...)
			a.tabs.Refresh()
		}()

		go func() {
			pp, err := a.iw.u.EveUniverseService().GetPlanets(ctx, planets)
			if err != nil {
				slog.Error("solar system info: Failed to load planets", "solarSystem", solarSystemID, "error", err)
				planetsLabel.Text = ihumanize.Error(err)
				planetsLabel.Importance = widget.DangerImportance
				planetsLabel.Refresh()
				return
			}
			xx := slices.Collect(xiter.MapSlice(pp, NewEntityItemFromEvePlanet))
			planetsTab.Content = NewEntityListFromItems(a.iw.show, xx...)
			a.tabs.Refresh()
		}()

	}()
	return nil
}
