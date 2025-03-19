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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
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
	typeImage := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(fyne.NewSquareSize(renderIconUnitSize))
	a := &locationArea{
		corporation:     corporation,
		corporationLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
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
	o, err := a.iw.u.EveUniverseService().GetOrCreateEveLocationESI(ctx, locationID)
	if err != nil {
		return err
	}
	go func() {
		r, err := a.iw.u.EveImageService().InventoryTypeRender(o.Type.ID, renderIconPixelSize)
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
		go a.iw.showZoomWindow(o.Name, o.Type.ID, a.iw.u.EveImageService().InventoryTypeRender, a.w)
	}
	description := o.Type.Description
	if description == "" {
		description = o.Type.Name
	}
	desc := widget.NewLabel(description)
	desc.Wrapping = fyne.TextWrapWord
	a.tabs.Append(container.NewTabItem("Description", container.NewVScroll(desc)))
	if a.iw.u.IsDeveloperMode() {
		x := NewAtributeItem("EVE ID", o.ID)
		x.Action = func(_ any) {
			a.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := NewAttributeList([]AttributeItem{x}...)
		attributeList.ShowInfoWindow = a.iw.ShowEveEntity
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}
	el := NewEntityListFromItems(
		a.iw.Show,
		NewEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.Region.ToEveEntity(), ""),
		NewEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.ToEveEntity(), ""),
		NewEntityItemFromEveSolarSystem(o.SolarSystem),
	)
	locationTab := container.NewTabItem("Location", el)
	a.tabs.Append(locationTab)
	a.tabs.Select(locationTab)
	a.tabs.Refresh()
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load corp logo", "owner", o.Owner, "error", err)
			return
		}
		a.corporationLogo.Resource = r
		a.corporationLogo.Refresh()
	}()
	return nil
}
