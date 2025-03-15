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
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	kxlayout "github.com/ErikKalkoken/fyne-kx/layout"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type solarSystemArea struct {
	Content fyne.CanvasObject

	iw InfoWindow
	w  fyne.Window

	region        *kxwidget.TappableLabel
	constellation *kxwidget.TappableLabel
	logo          *canvas.Image
	name          *widget.Label
	security      *widget.RichText
	tabs          *container.AppTabs
}

func newSolarSystemArea(iw InfoWindow, solarSystemID int32, w fyne.Window) *solarSystemArea {
	region := kxwidget.NewTappableLabel("", nil)
	region.Truncation = fyne.TextTruncateEllipsis
	constellation := kxwidget.NewTappableLabel("", nil)
	constellation.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	s := float32(app.IconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(s))
	a := &solarSystemArea{
		region:        region,
		constellation: constellation,
		iw:            iw,
		logo:          logo,
		name:          name,
		security:      widget.NewRichText(),
		tabs:          container.NewAppTabs(),
		w:             w,
	}
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
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(solarSystemID)
		if err != nil {
			slog.Error("solar system info update failed", "solarSystem", solarSystemID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load solarSystem: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *solarSystemArea) load(solarSystemID int32) error {
	ctx := context.Background()
	o, err := a.iw.eus.GetOrCreateEveSolarSystemESIPlus(ctx, solarSystemID)
	if err != nil {
		return err
	}
	go func() {
		id, err := o.GetStarTypeID(ctx)
		if err != nil {
			return
		}
		r, err := a.iw.eis.InventoryTypeIcon(id, app.IconPixelSize)
		if err != nil {
			slog.Error("solar system info: Failed to load logo", "solarSystem", solarSystemID, "error", err)
			return
		}
		a.logo.Resource = r
		a.logo.Refresh()
	}()
	a.name.SetText(o.System.Name)
	a.region.SetText(o.System.Constellation.Region.Name)
	a.region.OnTapped = func() {
		a.iw.ShowEveEntity(o.System.Constellation.Region.ToEveEntity())
	}
	a.constellation.SetText(o.System.Constellation.Name)
	a.constellation.OnTapped = func() {
		a.iw.ShowEveEntity(o.System.Constellation.ToEveEntity())
	}
	iwidget.SetRichText(a.security, o.System.DisplayRichText()...)

	systemsLabel := widget.NewLabel("Loading...")
	systemsTab := container.NewTabItem("Stargates", systemsLabel)
	a.tabs.Append(systemsTab)
	go func() {
		ss, err := o.GetAdjacentSystems(ctx)
		if err != nil {
			slog.Error("solar system info: Failed to load adjacent systems", "solarSystem", solarSystemID, "error", err)
			systemsLabel.Text = ihumanize.Error(err)
			systemsLabel.Importance = widget.DangerImportance
			systemsLabel.Refresh()
			return
		}
		xx := slices.Collect(xiter.MapSlice(ss, NewEntityItemFromEveSolarSystem))
		systemsTab.Content = NewEntityListFromItems(a.iw.Show, xx...)
		a.tabs.Refresh()
	}()

	planetsLabel := widget.NewLabel("Loading...")
	planetsTab := container.NewTabItem("Planets", planetsLabel)
	a.tabs.Append(planetsTab)
	go func() {
		pp, err := o.GetPlanets(ctx)
		if err != nil {
			slog.Error("solar system info: Failed to load planets", "solarSystem", solarSystemID, "error", err)
			planetsLabel.Text = ihumanize.Error(err)
			planetsLabel.Importance = widget.DangerImportance
			planetsLabel.Refresh()
			return
		}
		xx := slices.Collect(xiter.MapSlice(pp, NewEntityItemFromEvePlanet))
		planetsTab.Content = NewEntityListFromItems(a.iw.Show, xx...)
		a.tabs.Refresh()
	}()

	stations := NewEntityListFromEntities(a.iw.Show, o.Stations...)
	a.tabs.Append(container.NewTabItem("Stations", stations))

	if len(o.Structures) > 0 {
		xx := slices.Collect(xiter.MapSlice(o.Structures, func(x *app.EveLocation) entityItem {
			return NewEntityItem(
				x.ID,
				x.Name,
				"Structure",
				Location,
			)
		}))
		structures := NewEntityListFromItems(a.iw.Show, xx...)
		note := widget.NewLabel("Only contains structures known through characters")
		note.Importance = widget.LowImportance
		a.tabs.Append(container.NewTabItem("Structures", container.NewBorder(
			nil,
			note,
			nil,
			nil,
			structures,
		)))
	}
	a.tabs.Refresh()
	return nil
}
