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
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// locationArea represents an area that shows public information about a character.
type locationArea struct {
	Content fyne.CanvasObject

	corporation     *kxwidget.TappableLabel
	corporationLogo *canvas.Image
	iw              InfoWindow
	name            *widget.Label
	tabs            *container.AppTabs
	typeImage       *kxwidget.TappableImage
	typeInfo        *kxwidget.TappableLabel
	w               fyne.Window
}

func newLocationArea(iw InfoWindow, locationID int64, w fyne.Window) *locationArea {
	name := widget.NewLabel("Loading...")
	name.Truncation = fyne.TextTruncateEllipsis
	typeInfo := kxwidget.NewTappableLabel("", nil)
	typeInfo.Truncation = fyne.TextTruncateEllipsis
	corporation := kxwidget.NewTappableLabel("", nil)
	corporation.Truncation = fyne.TextTruncateEllipsis
	typeImage := kxwidget.NewTappableImage(icon.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(fyne.NewSquareSize(renderIconUnitSize))
	a := &locationArea{
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icon.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		iw:              iw,
		name:            name,
		typeInfo:        typeInfo,
		typeImage:       typeImage,
		tabs:            container.NewAppTabs(),
		w:               w,
	}

	p := theme.Padding()
	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.typeInfo,
		),
		container.NewBorder(
			nil,
			nil,
			a.corporationLogo,
			nil,
			a.corporation,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.typeImage), nil, main)
	a.Content = container.NewBorder(top, nil, nil, nil, a.tabs)

	go func() {
		err := a.load(locationID)
		if err != nil {
			slog.Error("location info update failed", "characterID", locationID, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", ihumanize.Error(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	return a
}

func (a *locationArea) load(locationID int64) error {
	ctx := context.Background()
	o, err := a.iw.eus.GetOrCreateEveLocationESI(ctx, locationID)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.eis.InventoryTypeRender(o.Type.ID, renderIconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load portrait", "location", o, "error", err)
			return
		}
		a.typeImage.SetResource(r)
	}()
	a.name.SetText(o.Name)
	a.typeInfo.SetText(o.Type.Name)
	a.typeInfo.OnTapped = func() {
		a.iw.ShowEveEntity(o.Type.ToEveEntity())
	}
	a.corporation.SetText(o.Owner.Name)
	a.corporation.OnTapped = func() {
		a.iw.ShowEveEntity(o.Owner)
	}
	a.typeImage.OnTapped = func() {
		go a.iw.showZoomWindow(o.Name, o.Type.ID, a.iw.eis.InventoryTypeRender, a.w)
	}
	if s := o.Type.Description; s != "" {
		description := widget.NewLabel(o.Type.Description)
		description.Wrapping = fyne.TextWrapWord
		a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(description)))
	}
	el := NewEntityListFromItems(
		a.iw.Show,
		NewEntityItemFromEveEntity(o.SolarSystem.Constellation.Region.ToEveEntity(), ""),
		NewEntityItemFromEveEntity(o.SolarSystem.Constellation.ToEveEntity(), ""),
		NewEntityItemFromEveSolarSystem(o.SolarSystem),
	)
	locationTab := container.NewTabItem("Location", el)
	a.tabs.Append(locationTab)
	a.tabs.Select(locationTab)
	a.tabs.Refresh()
	go func() {
		r, err := a.iw.eis.CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load corp logo", "owner", o.Owner, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	return nil
}
