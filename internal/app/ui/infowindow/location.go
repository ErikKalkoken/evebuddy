package infowindow

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
)

// locationInfo shows public information about a character.
type locationInfo struct {
	widget.BaseWidget

	id        int64
	owner     *kxwidget.TappableLabel
	ownerLogo *canvas.Image
	iw        *InfoWindow
	name      *widget.Label
	tabs      *container.AppTabs
	typeImage *kxwidget.TappableImage
	typeInfo  *kxwidget.TappableLabel
}

func newLocationInfo(iw *InfoWindow, id int64) *locationInfo {
	typeInfo := kxwidget.NewTappableLabel("", nil)
	typeInfo.Wrapping = fyne.TextWrapWord
	owner := kxwidget.NewTappableLabel("", nil)
	owner.Wrapping = fyne.TextWrapWord
	typeImage := kxwidget.NewTappableImage(icons.BlankSvg, nil)
	typeImage.SetFillMode(canvas.ImageFillContain)
	typeImage.SetMinSize(fyne.NewSquareSize(renderIconUnitSize))
	a := &locationInfo{
		id:        id,
		owner:     owner,
		ownerLogo: iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize)),
		iw:        iw,
		name:      makeInfoName(),
		typeInfo:  typeInfo,
		typeImage: typeImage,
		tabs:      container.NewAppTabs(),
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *locationInfo) CreateRenderer() fyne.WidgetRenderer {
	go func() {
		err := a.load()
		if err != nil {
			slog.Error("location info update failed", "locationID", a.id, "error", err)
			a.name.Text = fmt.Sprintf("ERROR: Failed to load character: %s", a.iw.u.ErrorDisplay(err))
			a.name.Importance = widget.DangerImportance
			a.name.Refresh()
		}
	}()
	p := theme.Padding()
	main := container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(layout.NewCustomPaddedVBoxLayout(-2*p),
			a.name,
			a.typeInfo,
		),
		container.NewBorder(
			nil,
			nil,
			a.ownerLogo,
			nil,
			a.owner,
		),
	)
	top := container.NewBorder(nil, nil, container.NewVBox(a.typeImage), nil, main)
	c := container.NewBorder(top, nil, nil, nil, a.tabs)
	return widget.NewSimpleRenderer(c)
}

func (a *locationInfo) load() error {
	ctx := context.Background()
	o, err := a.iw.u.EveUniverseService().GetOrCreateLocationESI(ctx, a.id)
	if err != nil {
		return err
	}
	a.name.SetText(o.Name)
	a.typeInfo.SetText(o.Type.Name)
	a.typeInfo.OnTapped = func() {
		a.iw.ShowEveEntity(o.Type.ToEveEntity())
	}
	a.owner.SetText(o.Owner.Name)
	a.owner.OnTapped = func() {
		a.iw.ShowEveEntity(o.Owner)
	}
	a.typeImage.OnTapped = func() {
		go a.iw.showZoomWindow(o.Name, o.Type.ID, a.iw.u.EveImageService().InventoryTypeRender, a.iw.w)
	}
	go func() {
		r, err := a.iw.u.EveImageService().InventoryTypeRender(o.Type.ID, renderIconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load portrait", "location", o, "error", err)
			return
		}
		a.typeImage.SetResource(r)
	}()
	go func() {
		r, err := a.iw.u.EveImageService().CorporationLogo(o.Owner.ID, app.IconPixelSize)
		if err != nil {
			slog.Error("location info: Failed to load corp logo", "owner", o.Owner, "error", err)
			return
		}
		a.ownerLogo.Resource = r
		a.ownerLogo.Refresh()
	}()

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
			a.iw.w.Clipboard().SetContent(fmt.Sprint(o.ID))
		}
		attributeList := NewAttributeList(a.iw, []AttributeItem{x}...)
		attributesTab := container.NewTabItem("Attributes", attributeList)
		a.tabs.Append(attributesTab)
	}

	el := NewEntityListFromItems(
		a.iw.show,
		NewEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.Region.ToEveEntity(), ""),
		NewEntityItemFromEveEntityWithText(o.SolarSystem.Constellation.ToEveEntity(), ""),
		NewEntityItemFromEveSolarSystem(o.SolarSystem),
	)
	locationTab := container.NewTabItem("Location", el)
	a.tabs.Append(locationTab)

	if o.Variant() == app.EveLocationStation {
		servicesLabel := widget.NewLabel("Loading...")
		servicesTab := container.NewTabItem("Services", servicesLabel)
		a.tabs.Append(servicesTab)
		go func() {
			ss, err := a.iw.u.EveUniverseService().GetStationServicesESI(ctx, int32(a.id))
			if err != nil {
				slog.Error("Failed to fetch station services", "stationID", o.ID, "error", err)
				servicesLabel.SetText("ERROR: Failed to load")
				return
			}
			items := xslices.Map(ss, func(s string) entityItem {
				s2 := strings.ReplaceAll(s, "-", " ")
				titler := cases.Title(language.English)
				name := titler.String(s2)
				return NewEntityItem(0, "Service", name, infoNotSupported)
			})
			servicesTab.Content = NewEntityListFromItems(nil, items...)
			a.tabs.Refresh()
		}()
	}

	a.tabs.Select(locationTab)
	a.tabs.Refresh()
	return nil
}
