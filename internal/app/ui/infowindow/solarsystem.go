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
	security      *widget.Label
	tabs          *container.AppTabs
}

func newSolarSystemArea(iw InfoWindow, solarSystemID int32, w fyne.Window) *solarSystemArea {
	region := kxwidget.NewTappableLabel("", nil)
	region.Truncation = fyne.TextTruncateEllipsis
	constellation := kxwidget.NewTappableLabel("", nil)
	constellation.Truncation = fyne.TextTruncateEllipsis
	name := widget.NewLabel("")
	name.Truncation = fyne.TextTruncateEllipsis
	s := float32(defaultIconPixelSize) * logoZoomFactor
	logo := iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(s))
	a := &solarSystemArea{
		region:        region,
		constellation: constellation,
		iw:            iw,
		logo:          logo,
		name:          name,
		security:      widget.NewLabel(""),
		tabs:          container.NewAppTabs(),
		w:             w,
	}
	colums := kxlayout.NewColumns(100)
	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		a.name,
		widget.NewLabel("Solar System"),
		layout.NewSpacer(),
		container.New(colums, widget.NewLabel("Region"), a.region),
		container.New(colums, widget.NewLabel("Constellation"), a.constellation),
		container.New(colums, widget.NewLabel("Security"), a.security),
	)
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
	o, err := a.iw.eus.GetOrCreateEveSolarSystemESI2(ctx, solarSystemID)
	if err != nil {
		return err
	}
	go func() {
		typ, err := a.iw.eus.GetStarTypeESI(ctx, o.StarID)
		if err != nil {
			slog.Error("solar system info: Failed to load star", "solarSystem", solarSystemID, "error", err)
			return
		}
		r, err := a.iw.eis.InventoryTypeIcon(typ.ID, defaultIconPixelSize)
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
	a.security.Text = fmt.Sprintf("%.1f", o.System.SecurityStatus)
	a.security.Importance = o.System.SecurityType().ToImportance()
	a.security.Refresh()

	stations := NewEntityListFromEntities(a.iw.Show, o.Stations...)
	a.tabs.Append(container.NewTabItem("Stations", stations))
	xx := slices.Collect(xiter.MapSlice(o.Structures, func(x *app.EveLocation) EntityItem {
		return EntityItem{
			ID:       x.ID,
			Text:     x.Name,
			Category: "Structure",
			Variant:  Location,
		}
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
	a.tabs.Refresh()
	return nil
}
